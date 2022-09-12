Convergen TODO
==============

- [ ] Generate return value style converter by default.  
  ```go
  type Convergen interface {
    ModelToDomain(*Model) *Domain
  }

  func ModelToDomain(*Model) *Domain {}
  ```

- [ ] Generate zero-memory style converter as an option.  
  ```go
  // by "-z" cli flag
  //generate: go run github.com/reedom/convergen -z

  // convergen:style:zero-memory    // or on the interface
  type Convergen interface {
    // convergen:style:zero-memory  // or on a specific method
    ModelToDomain(*Model) *Domain
  }

  func ModelToDomain(*Domain, *Model) {}
  ```


- [ ] Generate a converter that can return an error that is determined by the signature  
  ```go
  type Convergen interface {
    ModelToDomain(*Model) (*Domain, error)
  }

  func ModelToDomain(*Model) (*Domain, error) {}
  ```
  ```go
  type Convergen interface {
    // convergen:style:zero-memory
    ModelToDomain(*Model) (*Domain, error)
  }

  func ModelToDomain(*Domain, *Model) error {}
  ```


- [ ] Generate a converter as medhod.  
  ```go
  type Convergen interface {
    // convergen:receiver:m
    ModelToDomain(*Model) *Domain
  }

  func (m *Model) ToDomain() *Domain {}
  ```

- [ ] Support embedded struct.  
  ```go
  type Model struct {
    OtherModel
  }

  interface convergen {
    ModelToDomain(*Model) *Domain
  }
  ```

- [ ] Support case-insensitive field matching.  
  ```go
  // by "--nocase" cli flag
  //generate: go run github.com/reedom/convergen --nocase

  // convergen:opt:nocase          // or on the interface
  type Convergen interface {
    // convergen:opt:nocase        // or on a specific method
    ModelToDomain(*Model) *Domain
  }
  ```

- [ ] Support field name mappings.  
  ```go
  type Convergen interface {
    // convergen:map ID UserID
    ModelToDomain(*Model) *Domain
  }

  func ToDomain(rhs *Model) *Domain {
    var lhs Domain
    lhs.UserID = rhs.ID
    ...
  }
  ```

- [ ] Support conversion functions.  
  ```go
  type Convergen interface {
    // convergen:with Atoi ID
    // convergen:with Atoi Timestamp TS
    ModelToDomain(*Model) *Domain
  }

  // convergen:convert Model.Code.*      // general functions can also have `convert` specification.
  func Atoi(s string) int { ... }

  // convergen:convert:type FooType BarType
  func ConvFoo(v FooType) BarType { ... }
  
  // convergen:convert:to BarType
  func ConvTo(v Pet) BarType { ... }
  
  func ToDomain(rhs *Model) *Domain {
    var lhs Domain
    lhs.ID = Atoi(rhs.ID)
    lhs.TS = Atoi(rhs.Timestamp)
    ...
  }
  ```

- [ ] Support getter method matching.  
  ```go
  // by "--match" cli flag, you can specify the matching order
  //generate: go run github.com/reedom/convergen --match getter,field

  // convergen:opt:match getter,field    // or on the interface
  type Convergen interface {
    // convergen:opt:match getter,field  // or on a specific method
    // convergen:map Timestamp TS 
    ModelToDomain(*Model) *Domain
  }

  func ToDomain(rhs *Model) *Domain {
    var lhs Domain
    lhs.UserID = rhs.UserID()
    lhs.TS = rhs.Timestamp()
    ...
  }
  ```

- [ ] Support skipping assignment.  
  ```go
  type Convergen interface {
    // convergen:skip Test.*             // with regexp
    ModelToDomain(*Model) *Domain
  }
  ```

- [ ] Support deep-copy for slice, etc.
  ```go
  type Convergen interface {
    // convergen:deepcopy photoURLs
    ModelToDomain(*model.Pet) *domain.Pet
  }
  ```

