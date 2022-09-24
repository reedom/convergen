TODO
====

High priority
-------------

- [ ] wrap `if src.xx != nil {` for pointer type field
  - Unsure which is better, automatically or with a nottation specifically.

May implement if there is strong demand
---------------------------------------

- [ ] tag match
- [ ] `:conv:type &lt;_func_> &lt;_src type_> [_to type_]` notation
  - it allows to specify a converter for type(s).
- [ ] `:conv:with &lt;_func_> &lt;_dst field_>` notation
  - it allows to specify a src-struct-to-field converter.
- [ ] copy recursively
- [ ] deep copy for slices
  - cannot cover all cases, IMHO.
- [ ] deep copy for maps
  - cannot cover all cases, IMHO.
