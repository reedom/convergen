```
func toModel(src *domain.User) (dst *model.User) {
  // pre: is `dst.Prop` not null?
  // pre: is copy between `Prop` defined?
  // pre: is there any converter for `Prop`?
  // pre: is there any mapper for `Prop`?
  dst.Prop.Name = dst.Prop.Name
  
  dst.Prop.State.Prop.Name = src.Prop.State.Prop.Name
  
  copyProp := func(dst, src *Prop) error {
    dst.Name = src.Name
  }
  
  copySlices := func(src []El) []El {
    if len(src) == 0 {
      return nil
    }
    list := make([]El, len(src))
    copy(list, src)
    return list
  }

  copySlices2 := func(src []El) []El {
    if len(src) == 0 {
      return nil
    }
    list := make([]El, len(src))
    copy(list, src)
    return list
  }
}

```
