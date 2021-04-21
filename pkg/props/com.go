package props

import (
	"fmt"
	"log"
	"strconv"
)

const (
	SCHEME      = "golem.com.scheme"
	PRICE_MODEL = "golem.com.pricing.model"

	LINEAR_COEFFS  = "golem.com.pricing.model.linear.coeffs"
	DEFINED_USAGES = "golem.com.usage.vector"
)

// BillingScheme enum.
type BillingScheme string

const (
	BillingSchemePAYU BillingScheme = "payu"
)

func (e BillingScheme) Validate() error {
	switch e {
	case BillingSchemePAYU:
		return nil
	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

type PriceModel string

const (
	PriceModelLINEAR PriceModel = "linear"
)

func (e PriceModel) Validate() error {
	switch e {
	case PriceModelLINEAR:
		return nil
	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

type Counter string

const (
	CounterTIME    Counter = "golem.usage.duration_sec"
	CounterCPU     Counter = "golem.usage.cpu_sec"
	CounterSTORAGE Counter = "golem.usage.storage_gib"
	CounterMAXMEM  Counter = "golem.usage.gib"
	CounterUNKNOWN Counter = ""
)

func (e Counter) Validate() error {
	switch e {
	case CounterTIME, CounterCPU, CounterSTORAGE, CounterMAXMEM, CounterUNKNOWN:
		return nil
	default:
		return fmt.Errorf("unknown enum value: %v", e)
	}
}

const (
	ComScheme     = "Scheme"
	ComPriceModel = "PriceModel"
)

type Com struct {
	Scheme     BillingScheme `field:"optional"`
	PriceModel PriceModel    `field:"optional"`
}

func (c *Com) Keys() map[string]string {
	return map[string]string{
		ComScheme:     SCHEME,
		ComPriceModel: PRICE_MODEL,
	}
}

type ComLinear struct {
	Com
	FixedPrice float32
	PriceFor   map[Counter]float32
}

func (cl *ComLinear) Keys() map[string]string {
	baseKeys := cl.Com.Keys()
	return baseKeys
}

func (cl *ComLinear) CustomMapping(props Props) error {
	if cl.PriceModel != PriceModelLINEAR {
		log.Fatal("expected linear pricing model")
	}
	_coeffs, ok := props[LINEAR_COEFFS]
	if !ok {
		return fmt.Errorf("missing key: '%v'", LINEAR_COEFFS)
	}
	_usages, ok := props[DEFINED_USAGES]
	if !ok {
		return fmt.Errorf("missing key: '%v'", DEFINED_USAGES)
	}
	coeffs, err := asList(_coeffs)
	if err != nil {
		return err
	}
	usages, err := asList(_usages)
	if err != nil {
		return err
	}
	// Pop from coeffs.
	if len(coeffs) == 0 {
		return fmt.Errorf("pop from empty list")
	}
	// Make sure there are coressponding values for input coeffs.
	if len(coeffs) < len(usages) {
		return fmt.Errorf("list index out of range")
	}

	_fixedPrice, coeffs := coeffs[len(coeffs)-1], coeffs[:len(coeffs)-1]
	fixedPrice, err := getFloat(_fixedPrice)
	if err != nil {
		return err
	}
	priceFor := make(map[Counter]float32)
	for i := range coeffs {
		coeff, err := getFloat(coeffs[i])
		if err != nil {
			return fmt.Errorf("could not convert string to float")
		}
		// Skip on zero coeffs.
		if coeff == 0 {
			continue
		}
		if len(usages) <= i {
			return fmt.Errorf("list index out of range")
		}
		priceFor[Counter(fmt.Sprintf("%v", usages[i]))] = float32(coeff)
	}
	cl.FixedPrice = float32(fixedPrice)
	cl.PriceFor = priceFor
	return nil
}

func getFloat(v string) (float64, error) {
	errMsg := "could not convert string to float"
	x, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf(errMsg)
	}
	return x, nil
}
