package customize_webhook

import (
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"smartLB/controllers"
)

type ReqHandler struct{}

func (t *ReqHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		controllers.Events <- event.GenericEvent{}
		resp.WriteHeader(http.StatusOK)
	default:
		resp.WriteHeader(http.StatusNotFound)
	}
}
