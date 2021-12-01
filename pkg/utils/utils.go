package utils

import (
	"net/url"
	"strings"
)

func HostnameToURL(hostname string) string {
	slicedHostname := getSlicedHostname(hostname)
	slicedHostname = append([]string{"www"}, slicedHostname...)

	u := url.URL{
		Scheme: "https",
		Host:   strings.Join(slicedHostname, "."),
	}

	return u.String()
}

func URLToHostname(inURL string) string {
	u, err := url.Parse(inURL)
	if err != nil {
		panic(err)
	}

	return getTopLevelHostname(u.Host)
}

func StripHostname(hostname string) string {
	return getTopLevelHostname(hostname)
}

func getTopLevelHostname(hostname string) string {
	return strings.Join(getSlicedHostname(hostname), ".")
}

func getSlicedHostname(hostname string) []string {
	hostname = strings.ReplaceAll(hostname, "https://", "")
	hostname = strings.ToLower(hostname)
	split := strings.Split(hostname, ".")
	return split[len(split)-2 : len(split)]
}
