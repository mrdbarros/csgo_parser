package utils

import (
	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
)

func GetRoundTime(p dem.Parser, roundStartTime float64, tickRate int) float64 {
	return GetCurrentTime(p, tickRate) - roundStartTime
}

func GetCurrentTime(p dem.Parser, tickRate int) float64 {
	currentFrame := p.CurrentFrame()
	return float64(currentFrame) / float64(tickRate)
}
