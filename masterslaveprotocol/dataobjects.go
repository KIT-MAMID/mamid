package masterslaveprotocol

import (
	"encoding/json"
	"io"
)

type HostPort struct {
	Hostname string
	Port uint
}

type Mongod struct {
	Port uint
	ReplSetName string
	Targets []HostPort
	CurrentError error
}

type MSPError interface { // I am using an interface instead of a struct as error so that it can be nil without having to use pointers
	Error() string
	encodeJson(w io.Writer) // Cant json encode interface directly so use this method for that
}

func NewMSPError(error_message string) MSPError {
	mspError := new(mspErrorImpl)
	mspError.ErrorMessage = error_message
	return mspError
}

func MSPErrorFromJson(r io.Reader) MSPError {
	var mspError mspErrorImpl
	json.NewDecoder(r).Decode(&mspError) //TODO Check decode error
	return &mspError
}

type mspErrorImpl struct {
	ErrorMessage string
}

func (e mspErrorImpl) Error() string {
	return e.ErrorMessage
}

func (e mspErrorImpl) encodeJson(w io.Writer) {
	json.NewEncoder(w).Encode(e)
}