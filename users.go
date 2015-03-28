package main

import (
	"code.google.com/p/gopass"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"net/url"
	"strconv"
)

type dispValue struct{ Display, Value string }

type userAccount struct {
	Schemas               []string `json:",omitempty"`
	UserName              string
	Id                    string                                                     `json:",omitempty"`
	Active                bool                                                       `json:",omitempty"`
	Emails, Groups, Roles []dispValue                                                `json:",omitempty"`
	Meta                  *struct{ Created, LastModified, Location, Version string } `json:",omitempty"`
	Name                  struct{ GivenName, FamilyName string }                     `json:",omitempty"`
	WksExt                *struct{ InternalUserType, UserStatus string }             `json:"urn:scim:schemas:extension:workspace:1.0,omitempty"`
	Password              string                                                     `json:",omitempty"`
}

type userList struct {
	Resources                              []userAccount
	ItemsPerPage, TotalResults, StartIndex uint
	Schemas                                []string `json:",omitempty"`
}

func ppUserAccount(a *userAccount) {
	log(linfo, "%v, %v %v, %v\n%v\n\n", a.UserName, a.Name.GivenName, a.Name.FamilyName,
		a.Emails[0].Value, a.Id)
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
		log(lerr, "Error getting users: %v\n", err)
		return
	}
	//log(linfo, "User response: %#v\n", users)
	for _, v := range users.Resources {
		ppUserAccount(&v)
	}
}

func getpwd(prompt string) string {
	if s, err := gopass.GetPass(prompt); err != nil {
		panic(err)
	} else {
		return s
	}
}

func getPassword() (pwd string) {
	for {
		pwd = getpwd("Password: ")
		if pwd == getpwd("Password again: ") {
			return
		}
		log(linfo, "Passwords didn't match. Try again.")
	}
}

func stringOrDefault(v, dfault string) string {
	if v != "" {
		return v
	}
	return dfault
}

func InitUserCmd(c *cli.Context, minArgs int, needed string) (args []string, authHdr string) {
	args = c.Args()
	if len(args) < minArgs {
		log(lerr, "%s must be specified\n", needed)
	} else {
		authHdr = authHeader()
	}
	return
}

type basicUser struct {
	Name, Given, Family, Email, Pwd string `yaml:",omitempty,flow"`
}

func addUser(u *basicUser, authHdr string) (err error) {
	acct := userAccount{UserName: u.Name, Schemas: []string{"urn:scim:schemas:core:1.0"}}
	acct.Password = u.Pwd
	acct.Name.FamilyName = stringOrDefault(u.Family, u.Name)
	acct.Name.GivenName = stringOrDefault(u.Given, u.Name)
	acct.Emails = []dispValue{{Value: stringOrDefault(u.Email, u.Name+"@example.com")}}
	var body string
	return httpReq("POST", tgtURL("jersey/manager/api/scim/Users"), InitHdrs("", authHdr), &acct, &body)
}

func cmdAddUserBulk(c *cli.Context) {
	args, authHdr := InitUserCmd(c, 1, "file name of new users")
	var newUsers []basicUser
	if authHdr == "" {
		return
	}
	if inYaml, err := getFile(".", args[0]); err != nil {
		log(lerr, "could not read file of bulk users: %v\n", err)
	} else if err := yaml.Unmarshal(inYaml, &newUsers); err != nil {
		log(lerr, "Error parsing new users: %v\n", err)
	} else {
		for k, v := range newUsers {
			if err := addUser(&v, authHdr); err != nil {
				log(lerr, "Error adding user, line %d, name %s: %v\n", k+1, v.Name, err)
			} else {
				log(linfo, "added user %s\n", v.Name)
			}
		}
	}
}

func getArgOrPassword(c *cli.Context) string {
	if args := c.Args(); len(args) > 1 {
		return args[1]
	}
	return getPassword()
}

func cmdAddUser(c *cli.Context) {
	args, authHdr := InitUserCmd(c, 1, "user name")
	if authHdr == "" {
		return
	}
	user := basicUser{Name: args[0], Given: c.String("given"), Family: c.String("family"),
		Email: c.String("email"), Pwd: getArgOrPassword(c)}
	if err := addUser(&user, authHdr); err != nil {
		log(lerr, "Error creating user: %v\n", err)
	} else {
		log(linfo, "User successfully added\n")
	}
}

func getUser(c *cli.Context) (acct userAccount, authHdr string, ok bool) {
	var args []string
	args, authHdr = InitUserCmd(c, 1, "user name")
	if authHdr == "" {
		return
	}
	vals, users := make(url.Values), userList{}
	vals.Set("filter", fmt.Sprintf("userName eq \"%s\"", args[0]))
	path := fmt.Sprintf("jersey/manager/api/scim/Users?%v", vals.Encode())
	if err := httpReq("GET", tgtURL(path), InitHdrs("", authHdr), nil, &users); err != nil {
		log(lerr, "Error getting user: %v\n", err)
		return
	}
	if len(users.Resources) != 1 {
		log(lerr, "expected to find one user with name \"%s\", but found %d\n", args[0], len(users.Resources))
		return
	}
	return users.Resources[0], authHdr, true
}

func cmdGetUser(c *cli.Context) {
	if acct, _, ok := getUser(c); ok {
		ppUserAccount(&acct)
	}
}

func cmdDelUser(c *cli.Context) {
	var output string
	if acct, authHdr, ok := getUser(c); ok {
		path := fmt.Sprintf("jersey/manager/api/scim/Users/%s", acct.Id)
		if err := httpReq("DELETE", tgtURL(path), InitHdrs("", authHdr), nil, output); err != nil {
			log(lerr, "Error deleting user: %v\n", err)
		} else {
			log(linfo, "User \"%s\" deleted\n", acct.UserName)
		}
	}
}

func cmdSetPassword(c *cli.Context) {
	acct, authHdr, ok := getUser(c)
	if !ok {
		return
	}
	path := fmt.Sprintf("jersey/manager/api/scim/Users/%s", acct.Id)
	patch := struct {
		Schemas  []string
		Password string
	}{[]string{"urn:scim:schemas:core:1.0"},
		getArgOrPassword(c)}
	hdrs := InitHdrs("", authHdr)
	hdrs["X-HTTP-Method-Override"] = "PATCH"
	hdrs["Content-Type"] = hdrs["Accept"]
	var output string
	if err := httpReq("POST", tgtURL(path), hdrs, &patch, &output); err != nil {
		log(lerr, "Error updating user: %v\n", err)
		ppJson(lerr, "Error response", output)
	} else {
		log(linfo, "User \"%s\" updated\n", acct.UserName)
	}
}
