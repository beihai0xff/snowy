package calculator

import (
	"fmt"
	"math"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

type simpleCalculator struct{}

const chartTypeLine = "line"

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
	case domain.ModelCollisionMotion:
		return computeCollisionMotion(params), nil
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
		domain.ModelCollisionMotion,
	}
}

func computeProjectile(params map[string]float64) *domain.ComputeResult {
	v0 := valueOrDefault(params, "v0", 20)
	angleDeg := valueOrDefault(params, "angle_deg", 0)
	h := valueOrDefault(params, "h", valueOrDefault(params, "height", 20))

	g := valueOrDefault(params, "g", 9.8)
	if g <= 0 {
		g = 9.8
	}

	angleRad := angleDeg * math.Pi / 180
	vy0 := v0 * math.Sin(angleRad)
	vx := v0 * math.Cos(angleRad)

	landingTime := (vy0 + math.Sqrt(math.Max(0, vy0*vy0+2*g*h))) / g
	if landingTime <= 0 {
		landingTime = math.Sqrt(math.Max(0.01, 2*h/g))
	}

	t := valueOrDefault(params, "t", landingTime)
	x := vx * t
	y := h + vy0*t - 0.5*g*t*t
	landingX := vx * landingTime
	targetX := valueOrDefault(params, "target_x", 40)
	series := make([][]float64, 0, 21)

	for i := range 21 {
		pointT := landingTime * float64(i) / 20
		series = append(series, []float64{
			vx * pointT,
			math.Max(0, h+vy0*pointT-0.5*g*pointT*pointT),
		})
	}

	return &domain.ComputeResult{
		Values: map[string]float64{
			"x":            x,
			"y":            y,
			"landing_time": landingTime,
			"landing_x":    landingX,
			"target_error": landingX - targetX,
		},
		Chart: &domain.ChartSpec{
			ChartType: chartTypeLine,
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
			ChartType: chartTypeLine,
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
			ChartType: chartTypeLine,
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
	// The browser preview uses normalized teaching units rather than SI-scale
	// astronomy values.  Keep the calculator aligned with that convention so the
	// UI does not show huge unreadable forces for generic prompts such as
	// “卫星绕地球运动”.
	centralMass := valueOrDefault(params, "central_mass", valueOrDefault(params, "m1", 8))
	satelliteMass := valueOrDefault(params, "satellite_mass", valueOrDefault(params, "m2", 1))
	radius := valueOrDefault(params, "orbit_radius", valueOrDefault(params, "r", 3.6))
	tangentialSpeed := valueOrDefault(params, "tangential_speed", 2.25)
	gravityStrength := valueOrDefault(params, "gravitational_strength", 10)

	if radius <= 0 {
		radius = 3.6
	}

	gravityForce := gravityStrength * centralMass * satelliteMass / (radius * radius)
	centripetalNeed := satelliteMass * tangentialSpeed * tangentialSpeed / radius
	orbitalBalance := gravityForce / math.Max(0.001, centripetalNeed)
	period := 2 * math.Pi * radius / math.Max(0.001, tangentialSpeed)

	return &domain.ComputeResult{
		Values: map[string]float64{
			"gravity_force":     gravityForce,
			"centripetal_need":  centripetalNeed,
			"orbital_balance":   orbitalBalance,
			"orbit_period_demo": period,
		},
	}
}

func computeCollisionMotion(params map[string]float64) *domain.ComputeResult {
	m1 := valueOrDefault(params, "m1", 1.5)
	m2 := valueOrDefault(params, "m2", 1)
	v1 := valueOrDefault(params, "v1", 5)
	v2 := valueOrDefault(params, "v2", -2)
	restitution := math.Max(0, math.Min(1, valueOrDefault(params, "restitution", 0.9)))

	totalMass := m1 + m2
	if totalMass <= 0 {
		totalMass = 1
	}

	newV1 := ((m1-restitution*m2)*v1 + (1+restitution)*m2*v2) / totalMass
	newV2 := ((m2-restitution*m1)*v2 + (1+restitution)*m1*v1) / totalMass
	initialMomentum := m1*v1 + m2*v2
	finalMomentum := m1*newV1 + m2*newV2
	initialEnergy := 0.5*m1*v1*v1 + 0.5*m2*v2*v2
	finalEnergy := 0.5*m1*newV1*newV1 + 0.5*m2*newV2*newV2

	return &domain.ComputeResult{Values: map[string]float64{
		"v1_after":          newV1,
		"v2_after":          newV2,
		"initial_momentum":  initialMomentum,
		"final_momentum":    finalMomentum,
		"initial_kinetic_e": initialEnergy,
		"final_kinetic_e":   finalEnergy,
	}}
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
