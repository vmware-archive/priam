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
	UserName, Id          string
	Active                bool
	Emails, Groups, Roles []dispValue
	Meta                  struct{ Created, LastModified, Location, Version string }
	Name                  struct{ GivenName, FamilyName string }
	WksExt                struct{ InternalUserType, UserStatus string } `json:"urn:scim:schemas:extension:workspace:1.0"`
	Password              string                                        `json:",omitempty"`
}

type userList struct {
	Resources                              []userAccount
	ItemsPerPage, TotalResults, StartIndex uint
	Schemas                                []string
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
	args, email, familyname, givenname := c.Args(), c.String("email"), c.String("familyname"), c.String("givenname")
	if len(args) < 1 {
		log(lerr, "user name must be specified\n")
	}
	acct := userAccount{UserName: args[0]}
	if acct.Password = args[1]; acct.Password == "" {
		log(linfo, "Password: ")
		acct.Password = string(gopass.GetPasswdMasked())
	}
	if familyname != "" {
		acct.Name.FamilyName = familyname
	}
	if givenname != "" {
		acct.Name.GivenName = givenname
	}
	if email != "" {
		acct.Emails = []dispValue{{Value: email}}
	}

	/*
		post to /jersey/manager/api/scim/Users"
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
	*/
}
