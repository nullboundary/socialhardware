package main

import (
	"bitbucket.org/cicadaDev/utils"
	"strings"
)

var tokenPrivateKey = []byte(`e&?kz,qIUnDWpnSuB_yr[vAEE}Z_(YU`) //TODO: lets make a new key and put this somewhere safer!

func accessToken(authString string, seeds ...string) bool {

	//1. Verify that the authentication token is correct.
	header := strings.Split(authString, " ")
	if len(header) < 2 {
		return false //header is malformed
	}
	authToken := header[1] //take the second value "api-key token"
	if ok, err := utils.VerifyToken(tokenPrivateKey, authToken, seeds...); !ok {

		if err != nil {
			utils.Check(err)
		}
		return false //verify failed
	}

	return true

}
