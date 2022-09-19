Convergen
=========

A type-to-type copy function code generator.

Notations
---------

| notation                                           | location         | summary                                                 |
|----------------------------------------------------|------------------|---------------------------------------------------------|
| :style &lt;`return` &#124; `arg`>                  | interface,method | Set the style of the assignee variable input/output.    |
| :match &lt;`name` &#124; `tag` &#124; `none`>      | interface,method | Set the field matcher algorithm.                        |
| :case                                              | interface,method | Set case-sensitive for name or tag match.               |
| :case:off                                          | interface,method | Set case-insensitive for name or tag match.             |
| :getter                                            | interface,method | Include getters for name or tag match (default).        |
| :getter:off                                        | interface,method | Exclude getters for name or tag match.                  |
| :stringer                                          | interface,method | Call String() if appropriate in name or tag match.      |
| :typecast                                          | interface,method | Allow type casting if appropriate in name or tag match. |
| :rcv &lt;_var_>                                    | method           | Specify to generate in method form.                     |
| :skip &lt;_dst field_>                             | method           | Specify field(s) to omit.                               |
| :map &lt;_src field_> &lt;_dst field_>             | method           | Specify field mapping rule.                             |
| :tag &lt;_src tag_> [_to tag_]                     | method           | Specify tag mapping rule.                               |
| :conv &lt;_func_> &lt;_src field_> [_to field_]    | method           | Specify a converter for field(s).                       |
| :conv:type &lt;_func_> &lt;_src type_> [_to type_] | method           | Specify a converter for type(s).                        |
| :conv:with &lt;_func_> &lt;_dst field_>            | method           | Specify a src-struct-to-field converter.                |
| :postprocess &lt;_func_>                           | method           | Specify a post-process func.                            |

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
func ToModel(src *domain.Pet) (dst *model.Pet) {
```

with error:

```go
func ToModel(src *domain.Pet) (dst *model.Pet, err error) {
```

with receiver:

```go
func (src *domain.Pet) ToModel() (dst *model.Pet) {
```

Examples of `arg` style.

basic:

```go
func ToModel(dst *model.Pet, src *domain.Pet) {
```

with error:

```go
func ToModel(dst *model.Pet, src *domain.Pet) (err error) {
```

with receiver:

```go
func (src *domain.Pet) ToModel(dst *model.Pet) {
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

algorithm = "name" | "tag" | "none"
```

__Examples__

With `name` match, the generator matches up with fields or getters names (and their types).

```go
package model

type User struct {
  ID   int
  Name string
}

package web

type User struct {
  id   int
  name string
}

func (u *User) ID() int {
  return u.id
}

// :match name 
type Convergen interface {
  ToModel(*User) *model.User
}
```

Convergen generates:

```go
func ToModel(src *User) (dst *model.User) {
  dst := &model.User{}
  dst.ID = src.ID()
  dst.Name = src.name
  
  return
}
```

With `tag` match, the generator matches up with tag and its first part (and their types).
Convergen also needs a `tag` notation.

```go
package model

type User struct {
  UserID   int      `spanner:"userId"`
  Name     string   `spanner:"name,omitempty"`
}

package web

type User struct {
  userID   int      `json:"userId"`
  name     string   `json:"name"`
}

func (u *User) UserID() int {
  return u.userID
}

// :match tag
// :tag json spanner
type Convergen interface {
  ToModel(*User) *model.User
}
```

Convergen generates:

```go
func ToModel(src *User) (dst *model.User) {
  dst := &model.User{}
  dst.ID = src.userID
  dst.Name = src.name
  
  return
}
```
