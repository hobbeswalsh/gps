package main

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// A CheckResult is the result of a check.
type CheckResult struct {
	status      string
	metric      float64
	description string
	tags        []string
}

type checker interface {
	Check(args ...string) (CheckResult, error)
}

// A PingChecker checks pings
type PingChecker struct {
	host string
}

// Check checks pings
func (p PingChecker) Check(args ...string) (CheckResult, error) {
	var cr CheckResult

	out, err := exec.Command("ping", "-nq", "-c", "4", "-i", ".2", p.host).Output()
	if err != nil {
		return CheckResult{}, errors.New("Ping failed")
	}

	outString := string(out)
	lossRe := regexp.MustCompile("([0-9+].[0-9+])%")
	loss, err := strconv.ParseFloat(lossRe.FindStringSubmatch(outString)[1], 64)
	if err != nil {
		return cr, errors.New("Could not parse packet loss")
	}
	// metricsRe := regexp.MustCompile("([0-9*].[0-9*])/([0-9*].[0-9*])/([0-9*].[0-9*])/([0-9*].[0-9*]) ms")
	metricsRe := regexp.MustCompile("[0-9.]*/([0-9.]*)/[0-9.]*/[0-9.]* ms")
	avg, err := strconv.ParseFloat(metricsRe.FindStringSubmatch(outString)[1], 64)
	if err != nil {
		return cr, errors.New("Could not parse round-trip time")
	}
	cr.metric = avg
	if loss > .5 {
		cr.status = "critical"
	} else {
		cr.status = "ok"
	}

	cr.description = fmt.Sprintf("Ping round-trip time to %s", p.host)
	cr.tags = []string{"ping", "network", "latency"}
	return cr, nil
}

func runCheck(interval int, c checker, rc chan CheckResult, as ...string) {
	for {
		time.Sleep(time.Duration(interval) * time.Second)
		cr, err := c.Check(as...)
		if err != nil {
			fmt.Println(err)
		} else {
			rc <- cr
		}
	}
}

func loopChecks(cs []checker) {
	z := make(chan CheckResult)
	for _, checker := range cs {
		go runCheck(10, checker, z)
	}
	for {
		select {
		case rez := <-z:
			fmt.Println(rez)
		}
	}
}

func main() {
	checks := []checker{
		PingChecker{"www.google.com"},
		PingChecker{"www.yahoo.com"},
		PingChecker{"www.reddit.com"},
	}
	loopChecks(checks)
}
