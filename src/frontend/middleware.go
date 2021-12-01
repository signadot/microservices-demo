// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net/http"
	"math/rand"
	"strings"
	"strconv"
	"time"
	"github.com/patrickmn/go-cache"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ctxKeyLog struct{}
type ctxKeyRequestID struct{}

type logHandler struct {
	log  *logrus.Logger
	next http.Handler
}

type responseRecorder struct {
	b      int
	status int
	w      http.ResponseWriter
}

func (r *responseRecorder) Header() http.Header { return r.w.Header() }

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.w.Write(p)
	r.b += n
	return n, err
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.w.WriteHeader(statusCode)
}

func (lh *logHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID, _ := uuid.NewRandom()
	ctx = context.WithValue(ctx, ctxKeyRequestID{}, requestID.String())
	start := time.Now()
	rr := &responseRecorder{w: w}
	log := lh.log.WithFields(logrus.Fields{
		"http.req.path":   r.URL.Path,
		"http.req.method": r.Method,
		"http.req.id":     requestID.String(),
	})
	

	if v, ok := r.Context().Value(ctxKeySessionID{}).(string); ok {
		cachesizeKey := attribute.Key("cachesize")
		span := trace.SpanFromContext(ctx)
		cachesize := requestcache.ItemCount()
		span.SetAttributes(cachesizeKey.Int(cachesize))
		requestcache.Set(requestID.String(), v, cache.NoExpiration)
		log = log.WithField("session", v)
	}
	log.Debug("request started")
	defer func() {
		log.WithFields(logrus.Fields{
			"http.resp.took_ms": int64(time.Since(start) / time.Millisecond),
			"http.resp.status":  rr.status,
			"http.resp.bytes":   rr.b}).Debugf("request complete")
	}()

	ctx = context.WithValue(ctx, ctxKeyLog{}, log)
	r = r.WithContext(ctx)
	lh.next.ServeHTTP(rr, r)
}

func ensureSessionID(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sessionID string
		var min = 1
		var max = 100
		rnum := rand.Intn(max - min + 1) + min
		userAgent := r.UserAgent()
		if !strings.Contains(userAgent, "python") || (rnum <= PERCENTNORMAL && !(FORCEUSER == "1")) {
			c, err := r.Cookie(cookieSessionID)
			//u, _ := uuid.NewRandom()
			rsession := rand.Intn(100000 - 1000 + 1) + 1000
			sessionID = strconv.Itoa(rsession)
			if err == http.ErrNoCookie {
				http.SetCookie(w, &http.Cookie{
					Name:   cookieSessionID,
					Value:  sessionID,
					MaxAge: cookieMaxAge,
				})
			} else if err != nil {
				return
			} else {
				sessionID = c.Value
			}
		} else {
			sessionID = "20109"
			http.SetCookie(w, &http.Cookie{
				Name:   cookieSessionID,
				Value:  sessionID,
				MaxAge: cookieMaxAge,
			})
		}

		ctx := context.WithValue(r.Context(), ctxKeySessionID{}, sessionID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	}
}
