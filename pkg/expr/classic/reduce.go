package classic

import (
	"math"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/myback/grafana/pkg/expr/mathexp"
)

func nilOrNaN(f *float64) bool {
	return f == nil || math.IsNaN(*f)
}

func (cr classicReducer) ValidReduceFunc() bool {
	switch cr {
	case "avg", "sum", "min", "max", "count", "last", "median":
		return true
	case "diff", "diff_abs", "percent_diff", "percent_diff_abs", "count_not_null":
		return true
	}
	return false
}

// nolint: gocyclo
func (cr classicReducer) Reduce(series mathexp.Series) mathexp.Number {
	num := mathexp.NewNumber("", nil)
	num.SetValue(nil)

	if series.Len() == 0 {
		return num
	}

	value := float64(0)
	allNull := true

	vF := series.Frame.Fields[series.ValueIdx]

	switch cr {
	case "avg":
		validPointsCount := 0
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				value += *f
				validPointsCount++
				allNull = false
			}
		}
		if validPointsCount > 0 {
			value /= float64(validPointsCount)
		}
	case "sum":
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				value += *f
				allNull = false
			}
		}
	case "min":
		value = math.MaxFloat64
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				allNull = false
				if value > *f {
					value = *f
				}
			}
		}
		if allNull {
			value = 0
		}
	case "max":
		value = -math.MaxFloat64
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				allNull = false
				if value < *f {
					value = *f
				}
			}
		}
		if allNull {
			value = 0
		}
	case "count":
		value = float64(vF.Len())
		allNull = false
	case "last":
		for i := vF.Len() - 1; i >= 0; i-- {
			if f, ok := vF.At(i).(*float64); ok {
				if !nilOrNaN(f) {
					value = *f
					allNull = false
					break
				}
			}
		}
	case "median":
		var values []float64
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				allNull = false
				values = append(values, *f)
			}
		}
		if len(values) >= 1 {
			sort.Float64s(values)
			length := len(values)
			if length%2 == 1 {
				value = values[(length-1)/2]
			} else {
				value = (values[(length/2)-1] + values[length/2]) / 2
			}
		}
	case "diff":
		allNull, value = calculateDiff(vF, allNull, value, diff)
	case "diff_abs":
		allNull, value = calculateDiff(vF, allNull, value, diffAbs)
	case "percent_diff":
		allNull, value = calculateDiff(vF, allNull, value, percentDiff)
	case "percent_diff_abs":
		allNull, value = calculateDiff(vF, allNull, value, percentDiffAbs)
	case "count_non_null":
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if nilOrNaN(f) {
					continue
				}
				value++
			}
		}

		if value > 0 {
			allNull = false
		}
	}

	if allNull {
		return num
	}

	num.SetValue(&value)
	return num
}

func calculateDiff(vF *data.Field, allNull bool, value float64, fn func(float64, float64) float64) (bool, float64) {
	var (
		first float64
		i     int
	)
	// get the newest point
	for i = vF.Len() - 1; i >= 0; i-- {
		if f, ok := vF.At(i).(*float64); ok {
			if !nilOrNaN(f) {
				first = *f
				allNull = false
				break
			}
		}
	}
	if i >= 1 {
		// get the oldest point
		for i := 0; i < vF.Len(); i++ {
			if f, ok := vF.At(i).(*float64); ok {
				if !nilOrNaN(f) {
					value = fn(first, *f)
					allNull = false
					break
				}
			}
		}
	}
	return allNull, value
}

var diff = func(newest, oldest float64) float64 {
	return newest - oldest
}

var diffAbs = func(newest, oldest float64) float64 {
	return math.Abs(newest - oldest)
}

var percentDiff = func(newest, oldest float64) float64 {
	return (newest - oldest) / math.Abs(oldest) * 100
}

var percentDiffAbs = func(newest, oldest float64) float64 {
	return math.Abs((newest - oldest) / oldest * 100)
}
