package props

import "fmt"

const (
	INF_MEM       = "golem.inf.mem.gib"
	INF_STORAGE   = "golem.inf.storage.gib"
	INF_CORES     = "golem.inf.cpu.cores"
	TRANSFER_CAPS = "golem.activity.caps.transfer.protocol"
)

// BillingScheme enum.
type RuntimeType string

const (
	RuntimeTypeUNKNOWN    RuntimeType = ""
	RuntimeTypeWASMTIME   RuntimeType = "wasmtime"
	RuntimeTypeEMSCRIPTEN RuntimeType = "emscripten"
	RuntimeTypeVM         RuntimeType = "vm"
)

func (e RuntimeType) Validate() error {
	switch e {
	case RuntimeTypeUNKNOWN, RuntimeTypeWASMTIME, RuntimeTypeEMSCRIPTEN, RuntimeTypeVM:
		return nil

	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

type WasmInterface string

const (
	WasmInterfaceWASI_0         WasmInterface = "0"
	WasmInterfaceWASI_0preview1 WasmInterface = "0preview1"
)

func (e WasmInterface) Validate() error {
	switch e {
	case WasmInterfaceWASI_0, WasmInterfaceWASI_0preview1:
		return nil
	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

const (
	InfBaseMem       = "Mem"
	InfBaseRuntime   = "Runtime"
	InfBaseStorage   = "Storage"
	InfBaseTransfers = "Transfers"
)

type InfBase struct {
	Mem       float32
	Runtime   RuntimeType
	Storage   float32  `field:"optional"`
	Transfers []string `field:"optional"`
}

func (ib *InfBase) Keys() map[string]string {
	return map[string]string{
		InfBaseMem:       INF_MEM,
		InfBaseRuntime:   "golem.runtime.name",
		InfBaseStorage:   INF_STORAGE,
		InfBaseTransfers: TRANSFER_CAPS,
	}
}

const (
	InfVmCores = "Cores"
)

type InfVm struct {
	InfBase
	Cores int
}

func (iv *InfVm) Keys() map[string]string {
	baseMap := iv.InfBase.Keys()
	baseMap[InfVmCores] = INF_CORES
	return baseMap
}

func (iv *InfVm) CustomMapping(props Props) {
	iv.Runtime = RuntimeTypeVM
}

var InfVmKeys = (&InfVm{}).Keys()

const (
	ExeUnitRequestPackageUrl = "PackageUrl"
)

type ExeUnitRequest struct {
	PackageUrl string
}

func (eur *ExeUnitRequest) Keys() map[string]string {
	return map[string]string{
		ExeUnitRequestPackageUrl: "golem.srv.comp.task_package",
	}
}

type VmPackageFormat string

const (
	VmPackageFormatUNKNOWN       VmPackageFormat = ""
	VmPackageFormatGVMKIT_SQUASH VmPackageFormat = "gvmkit-squash"
)

func (e VmPackageFormat) Validate() error {
	switch e {
	case VmPackageFormatUNKNOWN, VmPackageFormatGVMKIT_SQUASH:
		return nil
	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

const (
	VMRequestPackageFormat = "PackageFormat"
)

type VMRequest struct {
	PackageFormat VmPackageFormat
}

func (vmr *VMRequest) Keys() map[string]string {
	return map[string]string{
		VMRequestPackageFormat: "golem.srv.comp.vm.package_format",
	}
}
