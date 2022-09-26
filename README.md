Convergen
=========

A type-to-type copy function code generator.

Notation Table
--------------

| notation                                  | location         | summary                                                                  |
|-------------------------------------------|------------------|--------------------------------------------------------------------------|
| :convergen                                | interface        | Mark the interface as a converter definition.                            |
| :match &lt;`name` &#124; `none`>          | interface,method | Set the field matcher algorithm (default: `name`).                       |
| :style &lt;`return` &#124; `arg`>         | interface,method | Set the style of the assignee variable input/output (default: `return`). |
| :recv &lt;_var_>                          | method           | Specify the source value as a receiver of the generated function.        |
| :reverse                                  | method           | Reverse copy direction. Might be useful with receiver form.              |
| :case                                     | interface,method | Set case-sensitive for name match (default).                             |
| :case:off                                 | interface,method | Set case-insensitive for name match.                                     |
| :getter                                   | interface,method | Include getters for name match.                                          |
| :getter:off                               | interface,method | Exclude getters for name match (default).                                |
| :stringer                                 | interface,method | Call String() if appropriate in name match.                              |
| :stringer:off                             | interface,method | Call String() if appropriate in name match (default).                    |
| :typecast                                 | interface,method | Allow type casting if appropriate in name match.                         |
| :typecast:off                             | interface,method | Suppress type casting if appropriate in name match (default).            |
| :skip &lt;_dst field_>                    | method           | Mark a destination field to skip copying.                                |
| :map &lt;_src_> &lt;_dst field_>          | method           | Map two pair as assign source and destination.                           |
| :conv &lt;_func_> &lt;_src_> [_to field_] | method           | Apply a converter to source value and assign the result to destination.  |
| :postprocess &lt;_func_>                  | method           | Call a function at the end.                           |

Sample
------

Generates function(s) that copy field to field between two types.

Write a generator code in a convention:

```go
//go:build convergen

package sample

import (
    "time"

    "github.com/sample/myapp/domain"
    "github.com/sample/myapp/storage"
)

//go:generate go run github.com/reedom/convergen@v0.3.2
type Convergen interface {
    // :typecast
    // :stringer
    // :map Created.UnixMilli() Created
    DomainToStorage(*domain.User) *storage.User
}
```

*covergen* will generate:

```go
// Code generated by github.com/reedom/convergen
// DO NOT EDIT.

package sample

import (
    "time"

    "github.com/sample/myapp/domain"
    "github.com/sample/myapp/storage"
)

func DomainToStorage(src *domain.User) (dst *storage.User) {
    dst = &storage.User{}
    dst.ID = int64(src.ID)
    dst.Name = src.Name
    dst.Status = src.Status.String()
    dst.Created = src.Created.UnixMilli()

    return
}
```

for these struct types:

```go
package domain

import (
    "time"
)

type User struct {
    ID      int
    Name    string
    Status  Status
    Created time.Time
}

type Status string

func (s Status) String() string {
    return string(s)
}
```

```go
package storage

type User struct {
    ID      int64
    Name    string
    Status  string
    Created int64
}
```

Installation and intro
----------------------

### Use as a Go generator

At under your Go project directory, install the module via `go get`:

```shell
$ go get -u github.com/reedom/convergen@latest
```

then write a generator as:

```go
//go:generate go run github.com/reedom/convergen@v0.3.2
type Convergen interface {
    …
}
````

### Use as a CLI command

Install the command via `go install`:

```shell
$ go install github.com/reedom/convergen@latest
```

then you can generate code by calling:

```shell
$ convergen any-codegen-defined-code.go
```

CLI help shows:

```shell
Usage: convergen [flags] <input path>

By default, the generated code is written to <input path>.gen.go

Flags:
  -dry
        dry run
  -log
        write log to <output path>.log
  -out string
        output file path
  -print
        print result code to STDOUT as well
```

Notations
---------

### `:convergen`

Mark the interface as a converter definition.

By default, Convergen look for only an interface named "Convergen" as a converter definition block.
By marking with `:convergen` notation, let Convergen recognize them. This is useful especially if you want
to define same name methods but having different receivers.

__Available locations__

interface

__Format__

```text
":convergen"
```

__Examples__

```go
// :convergen
type TransportConvergen interface {
    // :recv t
    ToDomain(*trans.Model) *domain.Model 
}

// :convergen
type PersistentConvergen interface {
    // :recv t
    ToDomain(*persistent.Model) *domain.Model 
}

```


### `:match <algorithm>`

Set the field matcher algorithm.

__Default__

`:match name`

__Available locations__

interface,method

__Format__

```text
":match" <algorithm>

algorithm = "name" | none"
```

__Examples__

With `name` match, the generator matches up with fields or getters names (and their types).

```go
package model

type User struct {
    ID   int
    Name string
}
```
```go
package web

type User struct {
    id   int
    name string
}

func (u *User) ID() int {
  return u.id
}
```
```go
// :match name 
type Convergen interface {
    ToStorage(*User) *storage.User
}
```

Convergen generates:

```go
func ToStorage(src *User) (dst *storage.User) {
    dst := &storage.User{}
    dst.ID = src.ID()
    dst.Name = src.name

    return
}
```

With `none` match, it only processes explicitly specified fields or getters via `:map` and `:conv`. 

### `:style <style>`

Set the style of the assignee variable input/output.

__Default__

`:style return`

__Available locations__

interface, method

__Format__

```text
":style" style

style = "arg" | "return"
```

__Examples__

Examples of `return` style.

basic:

```go
func ToStorage(src *domain.Pet) (dst *storage.Pet) {
```

with error:

```go
func ToStorage(src *domain.Pet) (dst *storage.Pet, err error) {
```

with receiver:

```go
func (src *domain.Pet) ToStorage() (dst *storage.Pet) {
```

Examples of `arg` style.

basic:

```go
func ToStorage(dst *storage.Pet, src *domain.Pet) {
```

with error:

```go
func ToStorage(dst *storage.Pet, src *domain.Pet) (err error) {
```

with receiver:

```go
func (src *domain.Pet) ToStorage(dst *storage.Pet) {
```

### `:recv <var>`

Specify the source value as a receiver of the generated function.

By the Go language specification, the receiver type must be defined in the same package to
where the Convergen generates.  
By convention, &lt;_var_> should be the same identifier with what the methods of the type defines.

__Default__

No receiver to be used.

__Available locations__

method

__Format__

```text
":recv" var

var = variable-identifier 
```

__Examples__

In the following example, it assumes `domain.User` is defined in other file under the same directory(package).  
And also assumes that other methods choose `u` as their receiver variable name. So the same comes.

To clarify, either is okay that to define types in a Convergen setup file or in separated files.  


```go
package domain

import (
    "github.com/sample/myapp/storage"
)

type Convergen interface {
    // :recv u
    ToStorage(*User) *storage.User  
}
```

Will have:

```go
package domain

import (
    "github.com/sample/myapp/storage"
)

type User struct {
    ID   int
    Name string
}

func (u *User) ToStorage() (dst *storage.User) {
    dst = &storage.User{}
    dst.ID = int64(u.ID)  
    dst.Name = u.Name

    return
}
```

### `:reverse`

Reverse copy direction. Might be useful with receiver form.  
To use `:reverse`, `:style arg` is required. (Otherwise it can't have any data source to copy from.)

__Default__

Copy in normal direction. In receiver form, receiver to a variable in argument.

__Available locations__

method

__Format__

```text
":reverse"
```

__Examples__

```go
package domain

import (
    "github.com/sample/myapp/storage"
)

type Convergen interface {
    // :style arg
    // :recv u
    // :reverse
    FromStorage(*User) *storage.User  
}
```

Will have:

```go
package domain

import (
    "github.com/sample/myapp/storage"
)

type User struct {
    ID   int
    Name string
}

func (u *User) FromStorage(src *storage.User) {
    u.ID = int(src.User)  
    u.Name = src.Name
}
```

### `:case` / `:case:off`

Control case-sensitive match or case-insensitive match.

The notification takes effect in `:match name`, `:getter` and `:skip`.  
Other notations, namely `:map`, `:conv`, keep case-sensitive match.

__Default__

":case"

__Available locations__

interface, method

__Format__

```go
":case"
":case:off"
```

__Examples__

// interface level notation makes ":case:off" as default.
// :case:off
type Convergen interface {
    // Turn on case-sensitive match for names.
    // :case
    ToUserModel(*domain.User) storage.User

    // Adopt the default, case-insensitive match in this case.
    ToCategoryModel(*domain.Category) storage.Category
}

### `:getter` / `:getter:off`

Include getters for name match.

__Default__

`:getter:off`

__Available locations__

interface, method

__Format__

```text
":getter"
":getter:off"
```

__Examples__

With those models:

```go
package domain

type User struct {
    name string
}

func (u *User) Name() string {
    return u.name
}
```

```go
package storage

type User struct {
    Name string
}
```

The default Convergen behaviour can't find the private `name` and won't notice the getter.  
So, with the following we'll get…

```go
type Convergen interface {
    ToStorageUser(*domain.User) *storage.User
}
````

```go
func ToStorageUser(src *domain.User) (dst *storage.User)
    dst = &storage.User{}
    // no match: dst.Name

    return
}
```

And with `:getter` we'll have…

```go
type Convergen interface {
    // :getter
    ToStorageUser(*domain.User) *storage.User
}
````

```go
func ToStorageUser(src *domain.User) (dst *storage.User)
    dst = &storage.User{}
    dst.Name = src.Name()

    return
}
```

Alternatively, you can get the same result with `:map`.  
This is worth to learn since `:getter` affects the entire method - `:map` allows you to get
the result selectively. 

```go
type Convergen interface {
    // :map Name() Name
    ToStorageUser(*domain.User) *storage.User
}
```

### `:stringer` / `:stringer:off`

Call String() if appropriate in name match.

__Default__

`:stringer:off`

__Available locations__

interface, method

__Format__

```text
":stringer"
":stringer:off"
```

__Examples__

With those models:

```go
package domain

type User struct {
    Status Status
}

type Status struct {
    status string
}

func (s Status) String() string {
    return string(s)
}

var (
    NotVerified = Status{"notVerified"}
    Verified    = Status{"verified"}
    Invalidated = Status{"invalidated"}
)
```

```go
package storage

type User struct {
    String string
}
```

For `status` field, Convergen has no idea how to assign `Status` type to `string` by default.  
Letting it to lookup `String()` methods by chance, it will employ while the method is appropriate to the assignee. 

```go
type Convergen interface {
    // :stringer
    ToStorageUser(*domain.User) *storage.User
}
````

```go
func ToStorageUser(src *domain.User) (dst *storage.User)
    dst = &storage.User{}
    dst.Status = src.Status.String()

    return
}
```

Alternatively, you can get the same result with `:map`.  
This is worth to learn since `:stringer` affects the entire method - `:map` allows you to get
the result selectively.

```go
type Convergen interface {
    // :map Status.String() Name
    ToStorageUser(*domain.User) *storage.User
}
```

### `:typecast`

Allow type casting if appropriate in name match.

__Default__

`:typecast:off`

__Available locations__

interface, method

__Format__

```text
":typecast"
":typecast:off"
```

__Examples__

With those models:

```go
package domain

type User struct {
    ID     int
    Name   string
    Status Status
}

type Status string

```

```go
package storage

type User struct {
    ID     int64  
    Name   string
    Status string
}
```

Convergen respect types strictly. So that it will give up copying fields if their types does not match.  
To note, Convergen relies on [types.AssignableTo(V, T Type) bool](https://pkg.go.dev/go/types#AssignableTo) method
from the standard packages. It means that the judge is done by the type system of Go itself, not by a dumb string type name match. 

Without `:typecast` turning on…
```go
type Convergen interface {
    ToDomainUser(*storage.User) *domain.User
}
````

We'll get:

```go
func ToDomainUser(src *storage.User) (dst *domain.User)
    dst = &domain.User{}
    // no match: dst.ID
    dst.Name = src.Name
    // no match: dst.Status

    return
}
```

With `:typecast` it turns to:

```go
func ToDomainUser(src *storage.User) (dst *domain.User)
    dst = &domain.User{}
    dst.ID = int(src.ID)
    dst.Name = src.Name
    dst.Status = domain.Status(src.Status)

    return
}
```

### `:skip <dst field>`

Mark a destination field to skip copying.  

A method can have multiple `:skip` lines that enable to skip multiple fields.
`:case` / `:case:off` affect to `:skip`.

__Available locations__

method

__Format__

```text
":skip" dst-field

dst-field    = field-path
field-path   = { identifier "." } identifier 
```

__Examples__

```go
type Convergen interface {
    // :skip Name
    // :skip Status
    // :skip Created
    ToStorage(*domain.User) *storage.User
}
```

### `:map <src> <dst field>`

Specify a field mapping rule.

When to use:
- copy value between fields having different names.
- want to assign a method's result value to a destination field.

A method can have multiple `:map` lines that enable to skip multiple fields.  

`:case:off` does not take effect to `:map`;
&lt;src> and &lt;dst field> are compared in case-sensitive manner.  

__Available locations__

method

__Format__

```text
":map" src dst-field

src                   = field-or-method-chain
dst-field             = field-path
field-path            = { identifier "." } identifier
field-or-getter-chain = { (identifier | getter) "." } (identifier | getter)
getter                = identifier "()"  
```

__Examples__

In the following, two ID-ish fields have the same meaning but different names.

```go
package domain

type User struct {
    ID int
    Name string
}
```
```go
package storage

type User struct {
    UserID int
    Name string
}
```

We can apply `:map` to connect them:
```go
type Convergen interface {
    // :map ID UserID
    ToStorage(*domain.User) *storage.User
}
```

```go
func ToStorage(src *domain.User) (dst *storage.User) {
    dst = storage.User{}
    dst.UserID = src.ID
    dst.Name = src.Name
    
    return
}
```

In another case below, `Status` type has a method to retrieve its raw value.

```go
package domain

type User struct {
    ID     int
    Name   string
    Status Status
}

type Status int

func (s Status) Int() int {
    return int(s)
}

var (
    NotVerified = Status(1)
    Verified    = Status(2)
    Invalidated = Status(3)
)
```
```go
package storage

type User struct {
    UserID int
    Name   string
    Status int
}
```
`:map` allows us to apply a method's return value to assign:

```go
type Convergen interface {
    // :map ID UserID
    // :map Status.Int() Status
    ToStorage(*domain.User) *storage.User
}
```

```go
func ToStorage(src *domain.User) (dst *storage.User) {
    dst = storage.User{}
    dst.UserID = src.ID
    dst.Name = src.Name
    dst.Status = src.Status.Int()

    return
}
```

The method's return value should be compatible with the destination field. If not, 
`:typecast` or `:stringer` might be a help. Or consider to use `:conv` notation instead.

### `:conv <func> <src> [dst field]`

Apply a converter to source value and assign the result to destination.

_func_ must accept _src_ value as the sole argument and returns either   
  a) sole value that is compatible with the _dst_, or  
  a) a pair of variables as (_dst_, error).   
For the latter case, the method definition should have `error` in return value(s). 

You can omit _dst field_ if the source and destination field paths are exactly the same.

`:case:off` does not take effect to `:map`;
&lt;src> and &lt;dst field> are compared in case-sensitive manner.

__Available locations__

method

__Format__

```text
":conv" func src [dst-field]

func                  = identifier
src                   = field-or-method-chain
dst-field             = field-path
field-path            = { identifier "." } identifier
field-or-getter-chain = { (identifier | getter) "." } (identifier | getter)
getter                = identifier "()"  
```

__Examples__

```go
package domain

type User struct {
    ID    int
    Email string
}
```
```go
package storage

type User struct {
    ID    int
    Email string
}
```

To store `Email` being encrypted, we can:

```go
import (
    // The referenced library should have imported anyhow.
    _ "github.com/sample/myapp/crypto"
)

type Convergen interface {
    // :conv crypto.Encrypt Email
    ToStorage(*domain.User) *storage.User
}
```

And we'll get:

```go
import (
    "github.com/sample/myapp/crypto"
    _ "github.com/sample/myapp/crypto"
)

func ToStorage(src *domain.User) (dst *storage.User) {
    dst = storage.User{}
    dst.ID = src.ID
    dst.Email = crypto.Encrypt(src.Email)

    return
}
```

Sometimes you want to use a converter function that returns `error`, too.  
In that case the converter method also should return `error`.

```go
import (
    // The referenced library should have imported anyhow.
    _ "github.com/sample/myapp/crypto"
)

type Convergen interface {
    // :conv crypto.Decrypt Email
    FromStorage(*storage.User) (*domain.User, error)
}
```

Goes to:

```go
import (
    "github.com/sample/myapp/crypto"
    _ "github.com/sample/myapp/crypto"
)

func ToStorage(src *storage.User) (dst *domain.User, err error) {
    dst = domain.User{}
    dst.ID = src.ID
    dst.Email, err = crypto.Decrypt(src.Email)
    if err != nil {
        return
    }

    return
}
```

### `:postprocess <func>`

Call a function at the end.

__Available locations__

method

__Format__

```text
":postprocess" func

func  = identifier
```

__Examples__

```go
type Convergen interface {
    // :postprocess
    FromStorage(*storage.User) *domain.User()
}

func doAnything(dst *domain.User(), src *storage.User) {
    …
}
```

Becomes:

```go
func FromStorage(src *storage.User) (dst *domain.User) {
    …
    doAnything(dst, src)
    
    return
}

func doAnything(dst *domain.User(), src *storage.User) {
    …
}
```

The function can return an error.  
With that, the method definition should have `error`
in return value(s).

```go
type Convergen interface {
    // :postprocess
    FromStorage(*storage.User) (*domain.User(), error)
}

func doAnything(dst *domain.User(), src *storage.User) error {
    …
}
```

Troubleshooting
---------------

TBD


Contributing
------------

I highly appreciate any kind of contributions from you!

- Report bugs.
- Report a use case that the current implementation seems not fulfill. 
- implement new features by making a pull-request.
- Add or improve the document or examples.
- Create a project's logo.
- Hit the ☆Star button.
- etc.
