package main

import (
	"net/http"
	"strings"
)

func StripPort(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func GetIPAddress(req *http.Request) string {
	hdrForwardedFor := req.Header.Get("X-Forwarded-For")
	if hdrForwardedFor == "" {
		return StripPort(req.RemoteAddr)
	}
	// X-Forwarded-For is potentially a list of addresses separated with ","
	parts := strings.Split(hdrForwardedFor, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	// TODO: should return first non-local address
	return parts[0]
}

func GetHost(req *http.Request) string {
	url := req.Header.Get("X-Forwarded-Host")
	if url == "" {
		url = req.Host
	}
	return url
}

func AddProtocol(url, protocol string) string {
	if url == "" {
		return ""
	}
	protocol += "://"
	if strings.HasPrefix(strings.ToLower(url), protocol) {
		return url
	}
	return protocol + url
}

func ExtractReferer(req *http.Request) string {
	referers := req.Header["Referer"]
	if len(referers) == 0 {
		return ""
	}
	referer := referers[0]
	return referer[strings.LastIndex(referer, "/")+1:]
}
