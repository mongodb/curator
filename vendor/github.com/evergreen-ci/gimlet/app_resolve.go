package gimlet

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mongodb/grip"
	"github.com/urfave/negroni"
)

// Handler returns a handler interface for integration with other
// server frameworks.
func (a *APIApp) Handler() (http.Handler, error) {
	return a.getNegroni()
}

// Resolve processes the data in an application instance, including
// all routes and creats a mux.Router object for the application
// instance.
func (a *APIApp) Resolve() error {
	if a.isResolved {
		return nil
	}

	if a.router == nil {
		a.router = mux.NewRouter().StrictSlash(a.StrictSlash)
	}

	if err := a.attachRoutes(a.router, true); err != nil {
		return err
	}

	a.isResolved = true

	return nil
}

// getHander internal helper resolves the negorni middleware for the
// application and returns it in the form of a http.Handler for use in
// stitching together applications.
func (a *APIApp) getNegroni() (*negroni.Negroni, error) {
	if err := a.Resolve(); err != nil {
		return nil, err
	}

	n := negroni.New()
	for _, m := range a.middleware {
		n.Use(m)
	}
	n.UseHandler(a.router)

	return n, nil
}

func (a *APIApp) attachRoutes(router *mux.Router, addPrefix bool) error {
	catcher := grip.NewCatcher()
	for _, route := range a.routes {
		if !route.IsValid() {
			catcher.Add(fmt.Errorf("%s is not a valid route, skipping", route.route))
			continue
		}

		var methods []string
		for _, m := range route.methods {
			methods = append(methods, strings.ToLower(m.String()))
		}

		handler := getRouteHandlerWithMiddlware(a.wrappers, route.handler)
		if route.version > 0 {
			versionedRoute := getVersionedRoute(addPrefix, a.prefix, route.version, route.route)
			router.Handle(versionedRoute, handler).Methods(methods...)
		}

		if route.version == a.defaultVersion {
			route.route = getDefaultRoute(addPrefix, a.prefix, route.route)
			router.Handle(route.route, handler).Methods(methods...)
		}
	}

	return catcher.Resolve()
}

func getRouteHandlerWithMiddlware(mws []Middleware, route http.Handler) http.Handler {
	if len(mws) == 0 {
		return route
	}

	n := negroni.New()
	for _, m := range mws {
		n.Use(m)
	}
	n.UseHandler(route)
	return n
}

func getVersionedRoute(addPrefix bool, prefix string, version int, route string) string {
	if !addPrefix {
		prefix = ""
	}

	if strings.HasPrefix(route, prefix) {
		if prefix == "" {
			return fmt.Sprintf("/v%d%s", version, route)
		}
		route = route[len(prefix):]
	}

	return fmt.Sprintf("%s/v%d%s", prefix, version, route)
}

func getDefaultRoute(addPrefix bool, prefix, route string) string {
	if !addPrefix {
		return route
	}

	if strings.HasPrefix(route, prefix) {
		return route
	}
	return prefix + route
}
