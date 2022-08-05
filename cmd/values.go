package cmd

import (
	"fmt"
	"strconv"
)

type float64Flag struct {
	Value *float64
}

func (f *float64Flag) String() string {
	if f.Value == nil {
		return ""
	}

	return fmt.Sprintf("%0.1f", *f.Value)
}

func (f *float64Flag) Set(value string) error {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	f.Value = &v

	return nil
}

func (f *float64Flag) Type() string {
	return "float64"
}
