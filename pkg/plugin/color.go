package plugin

import (
	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	resource2 "k8s.io/apimachinery/pkg/api/resource"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "unsafe"
)

var durationRound = (sprig.GenericFuncMap()["durationRound"]).(func(duration interface{}) string)

func colorKeyword(phase string) string {
	/* covers ".status.phase", ".status.state", ".status.reason", pod QOS,
		   for various types, e.g. pod, pv, pvc, svc, ns, etc ...

		Here a generic method is used since this can be used to cover unknown CRDs that follows conventions as well.
		This also helps with maintaining the list
	    E.g.:
	    * acme.cert-manager.io/v1alpha2 Order: .status.state: valid
	    * pvc: .status.phase Bound
	*/
	switch phase {
	case "Running", "Succeeded", "Active", "Available", "Bound", "valid", "Guaranteed", "Completed":
		return color.GreenString(phase)
	case "Pending", "Released", "Burstable":
		return color.YellowString(phase)
	case "Failed", "Unknown", "Terminating", "Evicted", "BestEffort", "OOMKilled", "ContainerCannotRun", "Error":
		return color.New(color.FgRed, color.Bold).Sprintf(phase)
	default:
		return phase
	}
}

func colorAgo(kubeDate string) string {
	t, _ := time.ParseInLocation("2006-01-02T15:04:05Z", kubeDate, time.UTC)
	duration := time.Since(t).Round(time.Second)
	return colorDuration(duration)
}

func ago(t time.Time) string {
	duration := time.Since(t).Round(time.Second)
	return durationRound(duration.String())
}

func colorDuration(duration time.Duration) string {
	str := durationRound(duration.String())
	if duration < time.Minute*5 {
		return color.RedString(str)
	}
	if duration < time.Hour {
		return color.YellowString(str)
	}
	if duration < time.Hour*24 {
		return color.MagentaString(str)
	}
	return str
}

func colorBool(cond bool, str string) string {
	if cond {
		return color.GreenString(str)
	} else {
		return color.New(color.FgRed, color.Bold).Sprintf(str)
	}
}

func colorExitCode(exitCode int) string {
	switch exitCode {
	case 0:
		return strconv.Itoa(exitCode)
	default:
		return color.RedString("%d", exitCode)
	}
}

func markRed(regex, s string) string {
	return markWithColor(regex, s, color.RedString)
}

func markYellow(regex, s string) string {
	return markWithColor(regex, s, color.YellowString)
}

func markGreen(regex, s string) string {
	return markWithColor(regex, s, color.GreenString)
}

func markWithColor(regex string, s string, colorStringFunc func(format string, a ...interface{}) string) string {
	re := regexp.MustCompile(regex)
	var result []string
	for _, line := range strings.Split(s, "\n") {
		for _, match := range re.FindAllString(line, -1) {
			line = strings.Replace(line, match, colorStringFunc(match), 1)
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func redIf(cond bool, str string) string {
	if cond {
		return color.RedString(str)
	}
	return str
}

func redBoldIf(cond bool, str string) string {
	if cond {
		return color.New(color.FgRed, color.Bold).Sprintf(str)
	}
	return str
}

//go:linkname signame runtime.signame
func signame(sig uint32) string

func signalName(signal int64) string {
	return signame(uint32(signal))
}

func humanizeSI(unit string, input float64) string {
	return humanize.SIWithDigits(input, 1, unit)
}

func quantityToFloat64(str string) float64 {
	quantity, _ := resource2.ParseQuantity(str)
	return float64(quantity.MilliValue()) / 1000
}

func quantityToInt64(str string) int64 {
	quantity, _ := resource2.ParseQuantity(str)
	return quantity.Value()
}

func percent(x, y float64) float64 {
	return x / y * 100
}
