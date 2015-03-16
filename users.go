package main

import (
	//"fmt"
	"github.com/codegangsta/cli"
	//"strings"
)

/*

wks users (--max) (--filter)
wks user:
  get username 
  add username []attr:val
  update username []attr:val
  del rm username

*/

type dispValue struct { Display, Value string}

type userAccount struct {
	UserName, Id string
	Active bool
	Emails, Groups, Roles []dispValue
	Meta struct{ Created, LastModified, Location, Version string}
	Name struct { GivenName, FamilyName string}
	WksExt struct { InternalUserType, UserStatus string} `yaml:"urn:scim:schemas:extension:workspace:1.0"`
}

type userList struct {
	Resources []userAccount
	ItemsPerPage, TotalResults, StartIndex uint
	Schemas []string
}

func cmdUsers(c *cli.Context) {
	users := userList{}
	if err := getAuthnJson("jersey/manager/api/scim/Users?count=1000", "", &users); err != nil {
		log(lerr, "Error getting session token: %v\n", err)
		return
	}
	log(linfo, "User response: %v\n", users)
	for _, v := range users.Resources {
		log(linfo, "name: %v, %v %v\n type %v %v\n\n", v.UserName, v.Name.GivenName, 
			v.Name.FamilyName, v.WksExt.InternalUserType, v.WksExt.UserStatus)
	}
}

