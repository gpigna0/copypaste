package main

import (
	"fmt"
	"net/http"
	"time"
)

func handlerWrapper(h func(HTMLWriter, *http.Request, session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ws := HTMLWriter{Writer: w, Status: 200, HTMX: isHTMX(r)}

		if s, ex := sessions.session(r); !ex && r.URL.Path != "/login" {
			w.Header().Set("HX-Retarget", "body")
			ws.HTMX = true // Login does not need index template even if the request is not from HTMX
			sendTemplate(ws, "", "login", "./html/login.html")
		} else {
			s.revitalize()
			ws.HTMX = isHTMX(r)
			h(ws, r, s)
		}

		timeStamp := time.Now()
		latency := timeStamp.Sub(start)

		path := r.URL.Path
		statusColor := statusCodeColor(ws.Status)
		methodColor := methodColor(r.Method)
		htmxColor := htmxColor(ws.HTMX)
		resetColor := reset

		clientIP := r.RemoteAddr
		method := r.Method
		statusCode := ws.Status

		fmt.Printf("%v |%s %3d %s| %13v | %15s |%s %-7s %s|%s HTMX %s|\n%s\n",
			timeStamp.Format("2006/01/02 - 15:04:05"),
			statusColor, statusCode, resetColor,
			latency,
			clientIP,
			methodColor, method, resetColor,
			htmxColor, resetColor,
			path,
		)
	}
}

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

func statusCodeColor(s int) string {
	switch {
	case s >= http.StatusContinue && s < http.StatusOK:
		return white
	case s >= http.StatusOK && s < http.StatusMultipleChoices:
		return green
	case s >= http.StatusMultipleChoices && s < http.StatusBadRequest:
		return white
	case s >= http.StatusBadRequest && s < http.StatusInternalServerError:
		return yellow
	default:
		return red
	}
}

func methodColor(m string) string {
	switch m {
	case http.MethodGet:
		return blue
	case http.MethodPost:
		return cyan
	case http.MethodPut:
		return yellow
	case http.MethodDelete:
		return red
	case http.MethodPatch:
		return green
	case http.MethodHead:
		return magenta
	case http.MethodOptions:
		return white
	default:
		return reset
	}
}

func htmxColor(h bool) string {
	if h {
		return green
	} else {
		return red
	}
}
