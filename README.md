# Priam - a Go CLI for VMware Identity Manager
[![Build Status](https://travis-ci.org/vmware/priam.svg?branch=master)](https://travis-ci.org/vmware/priam/)

This is a Go program to interact with [VMware Identity Manager](http://www.vmware.com/products/identity-manager.html) product.
For now it mostly implements the administrative functions of IDM.

## Getting Started

1. If you haven't already, set up a Go workspace according to the [Go docs](http://golang.org/doc).
2. To follow the examples below you need to have `make` installed. 
3. If you want to view code coverage of the test suite with `make coverage` as described below, you need to have `awk` installed.
4. Setup GOPATH environment variable

	Then:
	```
	$ mkdir -p $GOPATH/src/github.com/vmware
	$ cd $GOPATH/src/github.com/vmware
	$ git clone git@github.com:vmware/priam.git
	$ cd priam
	```

### Build it, run it, install it

To build it and run it:

    $ make
    $ ./priam

You can also install the `priam` executable in the `bin` directory of your Go workspace:

    $ make install
    $ $GOPATH/bin/priam

### Test it
Run the testsuite:

    $ make test

Run the testsuite with code coverage (will open the browser with code coverage):

    $ make coverage

## Documentation

To list available commands, use:

    $ priam help

To get help on specific command (for example, 'target'):

    $ priam help target

or 

    $ priam target -h

## Examples

You need an IDM organization (like https://xxx.vmwareidentity.com)

Almost all commands require the user to be logged in as an IDM administrator to run API calls.
All examples below assume you have run the following commands before:

    $ priam target https://xxx.vmwareidentity.com
    $ priam login <admin username>
    Password: <enter the admin password>
    Access token saved

Note that you can also login with an OAuth 2.0 client_id and secret. In that case use:

    $ priam login -c <oauth2 client id>
    Secret: <the oauth2 secret>
    Access token saved

You can also login using an OAuth2 authorization code flow. This allows you to log in as any user, 
not just a user in the system domain as shown above. It launches an external browser and opens
a local http listener to receive the OAuth2 tokens. However, to use this login option this 
application must be registered as an OAuth2 client in the target tenant. A helper command is 
provided for this purpose. If you are logged in as an administrator using one of the previous
login options, you can run this command to register the OAuth2 client:

    $ priam client register

To see what options are registered (client_id, client_secret, scopes, etc), run:

    $ priam client register --help

After registration is completed, you can log in as a user from any domain supported by the 
tenant, and using authentication methods as specified by access policy:

    $ priam login -a
    
This sequence shows the steps involved in accessing an AWS environment for command line
access. Assumptions are that an OIDC identity provider has been created in the AWS IAM
service, an application has been created in the target VIDM tenant, and the same client
ID/audience value has been specified in the corresponding IAM role's trusted relationship,
the IAM OIDC identity provider, and the VIDM application.  In VIDM terms, the clientID is
the same as the audience value in IAM identity provider terms.  For this example, the
clientID/audience value will be "foo".

    $ priam target https://<vidm_tenant_url>
    $ priam login -a -i foo
    $ priam token aws <IAM Role ARN> -i foo -p aws-us
    
The IAM Role ARN value can be retrieved by navigating to an AWS IAM Role and looking at
the Summary page.  An AWS IAM Role would look something like: 

    arn:aws:iam::123456789012:role/MyRole 

### Users

Login as admin as shown above, then run:

    $ priam user list

This will list all users in the system.
You can also filter the users by their attributes:

    $ priam user list --filter 'username eq "test"'

The filter follows the SCIM standard: http://www.simplecloud.info/specs/draft-scim-api-00.html

You can add a local user:

    $ priam user add --email email@acme.com --family Travolta --given John jtravolta 'password'

You can also add a list of users defined in a YAML file:

    $ priam user load list-of-users.yaml
    $ cat list-of-users.yaml
    ---
    - {name: user1, given: User1, family: Family1, email: user1@acme.com, pwd: welcome1}
    - {name: user2, given: User2, family: Family2, email: user2@acme.com, pwd: welcome2}
    - {name: user3, given: User3, family: Family3, email: user3@acme.com, pwd: welcome3}

The users will be added with the "User" role.
To add a new local user "joe" as administrator, use:

    $ priam user add --email joe@acme.com --family Joe --given Joe joe 'password'
    $ priam role member Administrator joe

### Applications

To list applications:

    $ priam app list

To add an application to the catalog, you need to define a YAML manifest file that will contain the application information.

    $ priam app add my-app.yaml

An example of the YAML file for a simple web link:

    $ cat my-app.yaml
```yaml
applications:
- name: my-example-application
  workspace:
    packageVersion: '1.0'
    description: GitHub's Demo App
    catalogItemType: WebAppLink
    authInfo:
      type: WebAppLink
      targetUrl: "http://www.vmware.com"
```

To add an application and entitle users to it, you can specify the entitlement in the manifest, then add the application.
Note that if the application already exists, the application information will only be updated and its entitlements too.

    $ priam app add my-saml-app-entitled.yaml
    App "fannys-saml-app" added to the catalog
    Entitled group "ALL USERS" to app "fannys-saml-app".

    $ cat my-saml-app-entitled.yaml
```yaml
applications:
- name: fannys-saml-app
  workspace:
    packageVersion: '1.0'
    description: Demo App for GiHub with entitlements to all users
    entitleGroup: ALL USERS
    catalogItemType: Saml20
    attributeMaps:
      userName: "${user.userName}"
      firstName: "${user.firstName}"
      lastName: "${user.lastName}"
    accessPolicy: default_access_policy_set
    authInfo:
      type: Saml20
      validityTimeSeconds: 200
      nameIdFormat: urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified
      nameId: "${user.userName}"
      configureAs: "manual"
      audience: "https://test.fanny.audience"
      assertionConsumerServiceUrl: "https://test.fanny/a/{domainName}/acs?RelayState=http://mail.google.com/a/{domainName}"
      recipientName: "https://test.fanny/a/{domainName}/acs"
      attributes:
      - name: https://aws.amazon.com/SAML/Attributes/Role
        nameFormat: urn:oasis:names:tc:SAML:2.0:attrname-format:uri
        nameSpace: ''
        value: arn:aws:iam::646585102600:role/MyRole,arn:aws:iam::12345678:saml-provider/my-org
      - name: https://aws.amazon.com/SAML/Attributes/RoleSessionName
        nameFormat: urn:oasis:names:tc:SAML:2.0:attrname-format:uri
        nameSpace: ''
        value: "${user.userName}"
```

## Contributing

The priam project team welcomes contributions from the community. If you wish to contribute code and you have not
signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any
questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). For more detailed information,
refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Priam is available under the [Apache 2 license](LICENSE).
