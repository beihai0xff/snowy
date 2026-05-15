package service

import (
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

func (s *serviceImpl) generateNativePhysicsArtifact(
	sceneSpec *domain.SceneSpec,
	mode domain.RenderMode,
) (*domain.RenderArtifact, error) {
	props := cloneNumberMap(sceneSpec.DefaultProps)
	if props == nil {
		props = map[string]float64{}
	}

	ensureNativePhysicsProps(sceneSpec.SceneType, props)

	artifact := &domain.RenderArtifact{
		SceneType:  sceneSpec.SceneType,
		RenderMode: mode,
		RenderManifest: &domain.RenderManifest{
			Entry:         "snowy-native-physics-engine",
			Framework:     "snowy-native-physics-engine",
			Sandbox:       "react-native",
			RenderMode:    mode,
			MountSelector: "#snowy-native-physics-root",
			Dependencies:  []string{"rapier3d", "native-webgl", "react-canvas"},
			AllowedAPIs: []string{
				"requestAnimationFrame",
				"ResizeObserver",
				"CanvasRenderingContext2D",
				"WebGLRenderingContext",
				"WebGL2RenderingContext",
			},
			BlockedAPIs: []string{
				"fetch",
				"XMLHttpRequest",
				"localStorage",
				"sessionStorage",
				"indexedDB",
				"document.cookie",
				"WebSocket",
				"navigator.sendBeacon",
			},
			InitialProps: props,
		},
		CodeBundle: map[string]string{
			"README.md": "Native physics artifact. The web client renders this scene with the bundled Rapier 3D engine; code_bundle is retained only for API compatibility.",
		},
		ResultSummary: nativePhysicsSummary(sceneSpec),
	}

	if sceneSpec.Summary != "" {
		artifact.ResultSummary = sceneSpec.Summary + " 已切换为 Rapier 3D 原生物理引擎预览，参数变化将直接驱动本地仿真。"
	}

	return artifact, nil
}

func ensureNativePhysicsProps(sceneType string, props map[string]float64) {
	switch sceneType {
	case scenePhysicsForce3D:
		ensureForce3DProps(props)
	case scenePhysicsProjectile3D, scenePhysicsProjectile2D:
		ensureProjectileProps(props)
	case scenePhysicsOrbit3D:
		ensureOrbitProps(props)
	case scenePhysicsSpring3D:
		ensureSpringProps(props)
	case scenePhysicsCollision3D:
		ensureCollisionProps(props)
	default:
		ensureSharedNativeProps(props)
	}
}

func ensureSharedNativeProps(props map[string]float64) {
	if _, ok := props["view_dimension"]; !ok {
		props["view_dimension"] = 3
	}

	if _, ok := props["animation_speed"]; !ok {
		props["animation_speed"] = 1
	}

	if _, ok := props["trail_length"]; !ok {
		props["trail_length"] = 180
	}
}

func ensureProjectileProps(props map[string]float64) {
	ensureSharedNativeProps(props)

	if _, ok := props["v0"]; !ok {
		props["v0"] = 20
	}

	if _, ok := props["angle_deg"]; !ok {
		props["angle_deg"] = 45
	}

	if _, ok := props["t"]; !ok {
		props["t"] = 2
	}

	if _, ok := props["g"]; !ok {
		props["g"] = 9.8
	}

	if _, ok := props["view_dimension"]; !ok {
		props["view_dimension"] = 3
	}
}

func nativePhysicsSummary(sceneSpec *domain.SceneSpec) string {
	if strings.TrimSpace(sceneSpec.Summary) != "" {
		return sceneSpec.Summary + " 已切换为 Rapier 3D 原生物理引擎预览，参数变化将直接驱动本地仿真。"
	}

	if strings.TrimSpace(sceneSpec.Title) != "" {
		return sceneSpec.Title + " 已切换为 Rapier 3D 原生物理引擎预览。"
	}

	return "已切换为 Rapier 3D 原生物理引擎预览。"
}

func ensureForce3DProps(props map[string]float64) {
	ensureSharedNativeProps(props)

	if _, ok := props["m"]; !ok {
		props["m"] = 2
	}

	if _, ok := props["a"]; !ok {
		props["a"] = 3
	}

	if _, ok := props["view_dimension"]; !ok {
		props["view_dimension"] = 3
	}

	if _, ok := props["camera_yaw"]; !ok {
		props["camera_yaw"] = 0.55
	}

	if _, ok := props["camera_pitch"]; !ok {
		props["camera_pitch"] = 0.42
	}
}

func ensureOrbitProps(props map[string]float64) {
	ensureSharedNativeProps(props)

	defaults := map[string]float64{
		"central_mass":           8,
		"satellite_mass":         1,
		"orbit_radius":           3.6,
		"tangential_speed":       2.25,
		"eccentricity":           0.18,
		"gravitational_strength": 10,
		"trail_length":           240,
		"camera_yaw":             0.72,
		"camera_pitch":           0.54,
	}
	for key, value := range defaults {
		if _, ok := props[key]; !ok {
			props[key] = value
		}
	}
}

func ensureSpringProps(props map[string]float64) {
	ensureSharedNativeProps(props)

	defaults := map[string]float64{
		"k":            24,
		"m":            1.2,
		"x":            1.4,
		"damping":      0.18,
		"trail_length": 180,
		"camera_yaw":   0.6,
		"camera_pitch": 0.38,
	}
	for key, value := range defaults {
		if _, ok := props[key]; !ok {
			props[key] = value
		}
	}
}

func ensureCollisionProps(props map[string]float64) {
	ensureSharedNativeProps(props)

	defaults := map[string]float64{
		"m1":           1.5,
		"m2":           1,
		"v1":           4.5,
		"v2":           -2.5,
		"restitution":  0.9,
		"trail_length": 200,
		"camera_yaw":   0.45,
		"camera_pitch": 0.38,
	}
	for key, value := range defaults {
		if _, ok := props[key]; !ok {
			props[key] = value
		}
	}
}
