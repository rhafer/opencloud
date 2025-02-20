package middleware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/proxy/policy"
)

// SelectorCookie provides a middleware which
func SelectorCookie(optionSetters ...Option) func(next http.Handler) http.Handler {
	options := newOptions(optionSetters...)
	logger := options.Logger
	policySelector := options.PolicySelector

	return func(next http.Handler) http.Handler {
		return &selectorCookie{
			next:           next,
			logger:         logger,
			policySelector: policySelector,
		}
	}
}

type selectorCookie struct {
	next           http.Handler
	logger         log.Logger
	policySelector config.PolicySelector
}

func (m selectorCookie) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if m.policySelector.Regex == nil && m.policySelector.Claims == nil {
		// only set selector cookie for regex and claim selectors
		m.next.ServeHTTP(w, req)
		return
	}

	selectorCookieName := ""
	if m.policySelector.Regex != nil {
		selectorCookieName = m.policySelector.Regex.SelectorCookieName
	} else if m.policySelector.Claims != nil {
		selectorCookieName = m.policySelector.Claims.SelectorCookieName
	}

	// update cookie
	if oidc.FromContext(req.Context()) != nil {

		selectorFunc, err := policy.LoadSelector(&m.policySelector)
		if err != nil {
			m.logger.Err(err)
		}

		selector, err := selectorFunc(req)
		if err != nil {
			m.logger.Err(err)
		}

		cookie := http.Cookie{
			Name:  selectorCookieName,
			Value: selector,
			Path:  "/",
		}
		http.SetCookie(w, &cookie)
	}

	m.next.ServeHTTP(w, req)
}
