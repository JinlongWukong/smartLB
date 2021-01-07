package customize_webhook

import (
	"fmt"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"smartLB/controllers"
	"smartLB/controllers/ipam"
)

type ReqHandler struct{}

func (t *ReqHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case "POST":
		controllers.Events <- event.GenericEvent{}
		resp.WriteHeader(http.StatusOK)
	case "GET":
		resp.WriteHeader(http.StatusOK)
		fmt.Fprintf(resp, ipam.VipPool.String())
	default:
		resp.WriteHeader(http.StatusNotFound)
	}
}
