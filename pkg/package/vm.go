package pkg

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/hhio618/go-golem/pkg/props"
	"github.com/pkg/errors"
)

const (
	defaultRepoSrv  = "_girepo._tcp.dev.golem.network"
	fallbackRepoUrl = "http://yacn2.dev.golem.network:8000"
)

type vmConstrains struct {
	minMemGib     float32
	minStoargeGib float32
	cores         float32
}

func (v *vmConstrains) String() string {

	return fmt.Sprintf("(&%v\n\t%v\n\t%v",
		fmt.Sprintf("%v>=%v", props.InfVmKeys["mem"], v.minMemGib),
		fmt.Sprintf("%v>=%v", props.InfVmKeys["storage"], v.minStoargeGib),
		fmt.Sprintf("%v=%v", props.InfVmKeys["runtime"], string(props.RuntimeTypeVM)),
	)
}

type vmPackage struct {
	repoUrl     string
	imageHash   string
	constraints string
}

func (v *vmPackage) ResolveUrl() (string, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%v/image.%v.link", v.repoUrl,
			v.imageHash), nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("status not ok")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	imageUrl := string(bodyBytes)
	return fmt.Sprintf("hash:sha3:%v:%v", v.imageHash, imageUrl), nil
}

func (v *vmPackage) DecorateDemand(demand *props.DemandBuilder) error {
	imageUrl, err := v.ResolveUrl()
	if err != nil {
		return err
	}
	demand.Ensure(v.constraints)
	demand.Add(&props.VMRequest{
		ExeUnitRequest: props.ExeUnitRequest{
			PackageUrl: imageUrl,
		},
		PackageFormat: props.VmPackageFormatGVMKIT_SQUASH,
	})
	return nil
}
func Repo(imageHash string, minMemGib float32, minStoargeGib float32) Package {
	if minMemGib == 0 {
		minMemGib = 0.5
	}
	if minStoargeGib == 0 {
		minStoargeGib = 2.0
	}
	return &vmPackage{
		repoUrl:   resolveRepoSrv(defaultRepoSrv),
		imageHash: imageHash,
		constraints: (&vmConstrains{
			minMemGib:     minMemGib,
			minStoargeGib: minStoargeGib,
		}).String(),
	}
}

func resolveRepoSrv(repoSrv string) string {
	_, addrs, err := net.LookupSRV("", "tcp", repoSrv)
	if err != nil {
		fmt.Printf("err: %v", &PackageError{errors.Wrap(err, "could not resolve Golem package repository address")})
		return fallbackRepoUrl
	}
	if len(addrs) == 0 {
		fmt.Printf("err: %v", &PackageError{errors.New("golem package repository is currently unavailable")})
		return fallbackRepoUrl
	}
	// Selecting a random srv.
	rand.Seed(time.Now().Unix())
	srv := addrs[rand.Intn(len(addrs))]
	return fmt.Sprintf("http://%v:%v", srv.Target, srv.Port)

}
