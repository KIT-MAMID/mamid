package masterapi

import (
	"fmt"
	"net/http"
)

type SharedSecret struct {
	Secret string
}

func (m *MasterAPI) SecretGet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "Not implemented")
}

func (m *MasterAPI) SecretUpdate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "Not implemented")
}

func (m *MasterAPI) DumpDB(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "Not implemented")
}

// Accept multipart/form-data file uploads; file parameter: config
func (m *MasterAPI) RestoreDB(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "Not implemented")
}
