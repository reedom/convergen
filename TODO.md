convergen TODO
==============

- [ ] Generate return value style converter by default.  
  ```go
  interface Convergen {
    ModelToDomain(*Model) *Domain
  }

  func ModelToDomain(*Model) *Domain {}
  ```

- [ ] Generate zero-memory style converter as an option.  
  ```go
  // by "-z" cli flag
  //generate: go run github.com/reedom/convergen -z

  // convergen:style:zero-memory    // or on the interface
  interface Convergen {
    // convergen:style:zero-memory  // or on a specific method
    ModelToDomain(*Model) *Domain
  }

  func ModelToDomain(*Domain, *Model) {}
  ```


- [ ] Generate a converter that can return an error that is determined by the signature  
  ```go
  interface Convergen {
    ModelToDomain(*Model) (*Domain, error)
  }

  func ModelToDomain(*Model) (*Domain, error) {}
  ```
  ```go
  interface Convergen {
    // convergen:style:zero-memory
    ModelToDomain(*Model) (*Domain, error)
  }

  func ModelToDomain(*Domain, *Model) error {}
  ```


- [ ] Generate a converter as medhod.  
  ```go
  interface Convergen {
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

  interface Convergen {
    ModelToDomain(*Model) *Domain
  }
  ```

- [ ] Support case-insensitive field matching.  
  ```go
  // by "--nocase" cli flag
  //generate: go run github.com/reedom/convergen --nocase

  // convergen:opt:nocase          // or on the interface
  interface Convergen {
    // convergen:opt:nocase        // or on a specific method
    ModelToDomain(*Model) *Domain
  }
  ```

- [ ] Support field name mappings.  
  ```go
  interface Convergen {
    // convergen:map ID UserID
    ModelToDomain(*Model) *Domain
  }

  func ToDomain(rhs *Model) *Domain {
    var lhs Domain
    lhs.UserID = rhs.ID
    ...
  }
  ```

- [ ] Support convertion functions.  
  ```go
  interface Convergen {
    // convergen:with Atoi ID
    // convergen:with Atoi Timestamp TS
    ModelToDomain(*Model) *Domain
  }

  // convergen:convert Model.Code.*      // general functions can also have `convert` specification.
  func Atoi(s string) int { ... }

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
  interface Convergen {
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
  interface Convergen {
    // convergen:skip Test.*             // with regexp
    ModelToDomain(*Model) *Domain
  }
  ```
