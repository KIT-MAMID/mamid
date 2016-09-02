package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"net/http"
)

type MongodKeyfile struct {
	Content string `json:"content"`
}

type MongodbCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (m *MasterAPI) KeyfileGet(w http.ResponseWriter, r *http.Request) {

	var modelKeyfile model.MongodKeyfile

	tx := m.DB.Begin()
	defer tx.Rollback()

	if err := tx.Table("mongod_keyfiles").First(&modelKeyfile).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error querying keyfile: %s", err.Error())
		return
	}

	apiKeyfile := ProjectModelMongodKeyfileToAPIMongodKeyfile(modelKeyfile)

	if err := json.NewEncoder(w).Encode(apiKeyfile); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	return

}

func (m *MasterAPI) ManagementUserGet(w http.ResponseWriter, r *http.Request) {

	var modelUser model.MongodbCredential

	tx := m.DB.Begin()
	defer tx.Rollback()

	if err := tx.Table("mongodb_root_credentials").First(&modelUser).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error querying MAMID management user credential: %s", err.Error())
		return
	}

	apiUser := ProjectModelMongodbCredentialToAPIMongodbCredential(modelUser)

	if err := json.NewEncoder(w).Encode(apiUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	return

}

func ProjectModelMongodKeyfileToAPIMongodKeyfile(m model.MongodKeyfile) (out MongodKeyfile) {
	return MongodKeyfile{
		Content: m.Content,
	}
}

func ProjectModelMongodbCredentialToAPIMongodbCredential(m model.MongodbCredential) (out MongodbCredential) {
	return MongodbCredential{
		Username: m.Username,
		Password: m.Password,
	}
}
