package calculator

import (
	"fmt"
	"math"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

type simpleCalculator struct{}

// NewSimpleCalculator 创建默认计算器实现。
func NewSimpleCalculator() Calculator {
	return &simpleCalculator{}
}

func (c *simpleCalculator) Compute(model domain.ModelType, params map[string]float64) (*domain.ComputeResult, error) {
	switch model {
	case domain.ModelProjectileMotion:
		return computeProjectile(params), nil
	case domain.ModelUniformAcceleration:
		return computeUniformAcceleration(params), nil
	case domain.ModelUniformMotion:
		return computeUniformMotion(params), nil
	case domain.ModelNewtonSecondLaw:
		return computeNewtonSecondLaw(params), nil
	case domain.ModelWorkEnergy:
		return computeWorkEnergy(params), nil
	case domain.ModelSpringOscillator:
		return computeSpringOscillator(params), nil
	case domain.ModelTwoBodyMotion:
		return computeTwoBodyMotion(params), nil
	}

	return nil, fmt.Errorf("unsupported physics model: %s", model)
}

func (c *simpleCalculator) SupportedModels() []domain.ModelType {
	return []domain.ModelType{
		domain.ModelProjectileMotion,
		domain.ModelUniformMotion,
		domain.ModelUniformAcceleration,
		domain.ModelNewtonSecondLaw,
		domain.ModelWorkEnergy,
		domain.ModelSpringOscillator,
		domain.ModelTwoBodyMotion,
	}
}

func computeProjectile(params map[string]float64) *domain.ComputeResult {
	v0 := valueOrDefault(params, "v0", 20)
	angleDeg := valueOrDefault(params, "angle_deg", 45)
	t := valueOrDefault(params, "t", 2)
	g := valueOrDefault(params, "g", 9.8)
	angleRad := angleDeg * math.Pi / 180
	x := v0 * math.Cos(angleRad) * t
	y := v0*math.Sin(angleRad)*t - 0.5*g*t*t
	series := make([][]float64, 0, 11)

	for i := range 11 {
		pointT := t * float64(i) / 10
		series = append(series, []float64{
			v0 * math.Cos(angleRad) * pointT,
			v0*math.Sin(angleRad)*pointT - 0.5*g*pointT*pointT,
		})
	}

	return &domain.ComputeResult{
		Values: map[string]float64{"x": x, "y": y},
		Chart: &domain.ChartSpec{
			ChartType: "line",
			Title:     "抛体轨迹图",
			XAxis:     domain.AxisSpec{Label: "x", Unit: "m"},
			YAxis:     domain.AxisSpec{Label: "y", Unit: "m"},
			Series:    []domain.SeriesSpec{{Name: "trajectory", Data: series}},
		},
		Warnings: collectWarnings(y),
	}
}

func computeUniformAcceleration(params map[string]float64) *domain.ComputeResult {
	x0 := valueOrDefault(params, "x0", 0)
	v0 := valueOrDefault(params, "v0", 0)
	a := valueOrDefault(params, "a", 2)
	t := valueOrDefault(params, "t", 5)
	x := x0 + v0*t + 0.5*a*t*t
	v := v0 + a*t
	series := make([][]float64, 0, 11)

	for i := range 11 {
		pointT := t * float64(i) / 10
		series = append(series, []float64{pointT, x0 + v0*pointT + 0.5*a*pointT*pointT})
	}

	return &domain.ComputeResult{
		Values: map[string]float64{"x": x, "v": v},
		Chart: &domain.ChartSpec{
			ChartType: "line",
			Title:     "位移-时间图像",
			XAxis:     domain.AxisSpec{Label: "t", Unit: "s"},
			YAxis:     domain.AxisSpec{Label: "x", Unit: "m"},
			Series:    []domain.SeriesSpec{{Name: "displacement", Data: series}},
		},
	}
}

func computeUniformMotion(params map[string]float64) *domain.ComputeResult {
	x0 := valueOrDefault(params, "x0", 0)
	v := valueOrDefault(params, "v", 5)
	t := valueOrDefault(params, "t", 5)
	x := x0 + v*t
	series := make([][]float64, 0, 11)

	for i := range 11 {
		pointT := t * float64(i) / 10
		series = append(series, []float64{pointT, x0 + v*pointT})
	}

	return &domain.ComputeResult{
		Values: map[string]float64{"x": x, "v": v},
		Chart: &domain.ChartSpec{
			ChartType: "line",
			Title:     "匀速直线运动图像",
			XAxis:     domain.AxisSpec{Label: "t", Unit: "s"},
			YAxis:     domain.AxisSpec{Label: "x", Unit: "m"},
			Series:    []domain.SeriesSpec{{Name: "position", Data: series}},
		},
	}
}

func computeNewtonSecondLaw(params map[string]float64) *domain.ComputeResult {
	m := valueOrDefault(params, "m", 2)
	a := valueOrDefault(params, "a", 3)

	return &domain.ComputeResult{Values: map[string]float64{"F": m * a}}
}

func computeWorkEnergy(params map[string]float64) *domain.ComputeResult {
	m := valueOrDefault(params, "m", 1)
	v := valueOrDefault(params, "v", 2)

	return &domain.ComputeResult{Values: map[string]float64{"kinetic_energy": 0.5 * m * v * v}}
}

func computeSpringOscillator(params map[string]float64) *domain.ComputeResult {
	k := valueOrDefault(params, "k", 20)
	m := valueOrDefault(params, "m", 1)
	x := valueOrDefault(params, "x", 0.2)

	return &domain.ComputeResult{
		Values: map[string]float64{
			"period":            2 * math.Pi * math.Sqrt(m/k),
			"elastic_potential": 0.5 * k * x * x,
		},
	}
}

func computeTwoBodyMotion(params map[string]float64) *domain.ComputeResult {
	const gravitationalConstant = 6.67430e-11

	m1 := valueOrDefault(params, "m1", 5.97e24)
	m2 := valueOrDefault(params, "m2", 7.35e22)
	r := valueOrDefault(params, "r", 3.84e8)

	return &domain.ComputeResult{
		Values: map[string]float64{
			"force": gravitationalConstant * m1 * m2 / (r * r),
		},
	}
}

func valueOrDefault(params map[string]float64, key string, fallback float64) float64 {
	if params == nil {
		return fallback
	}

	if value, ok := params[key]; ok {
		return value
	}

	return fallback
}

func collectWarnings(y float64) []string {
	if y >= 0 {
		return nil
	}

	return []string{"当前参数下物体已落回参考平面以下"}
}
