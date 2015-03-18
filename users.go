package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/howeyc/gopass"
	"net/url"
	"strconv"
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

type dispValue struct{ Display, Value string }

type userAccount struct {
	Schemas               []string `json:",omitempty"`
	UserName              string
	Id                    string                                                    `json:",omitempty"`
	Active                bool                                                      `json:",omitempty"`
	Emails, Groups, Roles []dispValue                                               `json:",omitempty"`
	Meta                  *struct{ Created, LastModified, Location, Version string } `json:",omitempty"`
	Name                  struct{ GivenName, FamilyName string }                    `json:",omitempty"`
	WksExt                *struct{ InternalUserType, UserStatus string }             `json:"urn:scim:schemas:extension:workspace:1.0,omitempty"`
	Password              string                                                    `json:",omitempty"`
}

type userList struct {
	Resources                              []userAccount
	ItemsPerPage, TotalResults, StartIndex uint
	Schemas                                []string `json:",omitempty"`
}

func cmdUsers(c *cli.Context) {
	count, filter, users, vals := c.Int("count"), c.String("filter"), userList{}, make(url.Values)
	if count == 0 {
		count = 1000
	}
	vals.Set("count", strconv.Itoa(count))
	if filter != "" {
		vals.Set("filter", filter)
	}
	path := fmt.Sprintf("jersey/manager/api/scim/Users?%v", vals.Encode())
	if err := getAuthnJson(path, "", &users); err != nil {
		log(lerr, "Error getting session token: %v\n", err)
		return
	}
	log(linfo, "User response: %#v\n", users)
	for _, v := range users.Resources {
		log(linfo, "name: %v, %v %v\n type %v %v\n\n", v.UserName, v.Name.GivenName,
			v.Name.FamilyName, v.WksExt.InternalUserType, v.WksExt.UserStatus)
	}
}

func cmdUser(c *cli.Context) {
	var err error
	args, email, familyname, givenname := c.Args(), c.String("email"), c.String("familyname"), c.String("givenname")
	if len(args) < 1 {
		log(lerr, "user name must be specified\n")
	}
	var authHdr string
	if authHdr, err = getAuthHeader(); err != nil {
		log(lerr, "Error getting access token: %v\n", err)
		return
	}
	acct := userAccount{UserName: args[0], Schemas: []string{"urn:scim:schemas:core:1.0"}}
	if len(args) < 2 {
		log(linfo, "Password: ")
		acct.Password = string(gopass.GetPasswdMasked())
	} else {
		acct.Password = args[1]
	}
	if familyname != "" {
		acct.Name.FamilyName = familyname
	} else {
		acct.Name.FamilyName = args[0]
	}

	if givenname != "" {
		acct.Name.GivenName = givenname
	} else {
		acct.Name.GivenName = args[0]
	}
	if email != "" {
		acct.Emails = []dispValue{{Value: email}}
	} else {
		acct.Emails = []dispValue{{Value: args[0] + "@example.com"}}
	}

	var body string

	if err := httpReq("POST", tgtURL("jersey/manager/api/scim/Users"),
		InitHdrMap("", authHdr), &acct, &body); err != nil {
		log(lerr, "Error creating user: %v\n", err)
		return
	}

	log(linfo, "reply %s", body)
}
