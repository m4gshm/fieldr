# Fieldr

Generator of various enumerated constants, types, functions based on
structure fields and their tags.

## Supported commands

- enum-const - generate constants based on template applied to struct
  fields.

- get-set - generates getters, setters for a structure type.

- builder - generates builder API of a structure type.

- as-map - generates method or functon that converts the struct type to
  a map.

## Installation

``` console
go install github.com/m4gshm/fieldr@latest
```

or

``` console
go install github.com/m4gshm/fieldr@HEAD
```

## Enum constant example

source `entity.go`:

``` go
package enum_const

//go:generate fieldr -type Entity enum-const -val tag.json -list jsons
type Entity struct {
    Id   int    `json:"id"`
    Name string `json:"name"`
}
```

then running this command in the same directory:

``` console
go generate .
```

will be generated `entity_fieldr.go` file with the next content:

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package enum_const

const (
    entityJsonId   = "id"
    entityJsonName = "name"
)

func jsons() []string {
    return []string{entityJsonId, entityJsonName}
}
```

this consist of constants based on the `json` tag contents and the
method `jsons` that enumerates these constants.

To get extended help of the command, use the following:

``` console
fieldr enum-const help
```

### Example of generating ORM elements:

source `entity.go`:

``` go
package enum_const_db

//go:generate fieldr -type Entity
//go:fieldr enum-const -name "'col' + field.name" -val "tag.db" -flat Versioned -type column -list . -ref-access .
//go:fieldr enum-const -name "'pk' + field.name" -val "tag.db" -include "tag.pk != nil" -type column -list pk

type Entity struct {
    BaseEntity
    Versioned *VersionedEntity
    Name string `db:"name"`
}

type BaseEntity struct {
    ID int32 `db:"id" pk:""`
}

type VersionedEntity struct {
    Version int64 `db:"version"`
}
```

generated `entity_fieldr.go`:

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package enum_const_db

type column string

const (
    colID      column = "id"
    colVersion column = "version"
    colName    column = "name"
    pkID       column = "id"
)

func columns() []column {
    return []column{colID, colVersion, colName}
}

func (s *Entity) ref(f column) any {
    if s == nil {
        return nil
    }
    switch f {
    case colID:
        return &s.BaseEntity.ID
    case colVersion:
        if v := s.Versioned; v != nil {
            return &v.Version
        }
    case colName:
        return &s.Name
    }
    return nil
}

func pk() []column {
    return []column{pkID}
}
```

#### explanation of used args:

- name *"{{ join \\"col\\" field.name }}"* - defines constant names as
  'col' appended by the associated field name.

- val *"tag.db"* - defines the value of a constant as a copy of the `db`
  tag of the associated field name.

- flat *Versioned* - also uses the `VersionedEntity` fields as constants
  source type in addition to the base `Entity` type.

- type *column* - adds the `column` type, and uses it as the type of the
  generated constants.

- list *.* - generates the `columns` method that returns constant
  values. It can be used to build sql queries like INSERT, SELECT.

- ref-access *.* - generates the `ref` method that provides access to
  the filed values, returns a reference pointing to the field associated
  with the constant. The method can be used in conjunction with Row.Scan
  from sql package.

## Get-set usage example

source `entity.go`

``` go
package get_set

import "time"

//go:generate fieldr -type Entity get-set

type BaseEntity[ID any] struct {
    id ID
}

type Entity[ID any] struct {
    *BaseEntity[ID]
    name    string
    surname string
    ts      time.Time
}
```

``` console
go generate .
```

generated `entity_fieldr.go`

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package get_set

import "time"

func (e *Entity[ID]) Id() ID {
    if e != nil {
        if be := e.BaseEntity; be != nil {
            return be.id
        }
    }

    var no ID
    return no
}

func (e *Entity[ID]) SetId(id ID) {
    if e != nil {
        if be := e.BaseEntity; be != nil {
            be.id = id
        }
    }
}

func (e *Entity[ID]) Name() string {
    if e != nil {
        return e.name
    }

    var no string
    return no
}

func (e *Entity[ID]) SetName(name string) {
    if e != nil {
        e.name = name
    }
}

func (e *Entity[ID]) Surname() string {
    if e != nil {
        return e.surname
    }

    var no string
    return no
}

func (e *Entity[ID]) SetSurname(surname string) {
    if e != nil {
        e.surname = surname
    }
}

func (e *Entity[ID]) Ts() time.Time {
    if e != nil {
        return e.ts
    }

    var no time.Time
    return no
}

func (e *Entity[ID]) SetTs(ts time.Time) {
    if e != nil {
        e.ts = ts
    }
}
```

## Builder usage example

source `entity.go`

``` go
package builder

//go:generate fieldr -type Entity builder -deconstructor .
type Entity[ID any] struct {
    *Model[ID]
    Name string
}

type Model[ID any] struct {
    ID        ID
    CreatedAt int64
    UpdatedAt int64
}
```

``` console
go generate .
```

generated `entity_fieldr.go`

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package builder

type EntityBuilder[ID any] struct {
    iD        ID
    createdAt int64
    updatedAt int64
    name      string
}

func NewEntityBuilder[ID any]() *EntityBuilder[ID] {
    return &EntityBuilder[ID]{}
}

func (b *EntityBuilder[ID]) Build() *Entity[ID] {
    if b == nil {
        return &Entity[ID]{}
    }
    return &Entity[ID]{
        Model: &Model[ID]{
            ID:        b.iD,
            CreatedAt: b.createdAt,
            UpdatedAt: b.updatedAt,
        },
        Name: b.name,
    }
}

func (b *EntityBuilder[ID]) ID(iD ID) *EntityBuilder[ID] {
    if b != nil {
        b.iD = iD
    }
    return b
}

func (b *EntityBuilder[ID]) CreatedAt(createdAt int64) *EntityBuilder[ID] {
    if b != nil {
        b.createdAt = createdAt
    }
    return b
}

func (b *EntityBuilder[ID]) UpdatedAt(updatedAt int64) *EntityBuilder[ID] {
    if b != nil {
        b.updatedAt = updatedAt
    }
    return b
}

func (b *EntityBuilder[ID]) Name(name string) *EntityBuilder[ID] {
    if b != nil {
        b.name = name
    }
    return b
}

func (e *Entity[ID]) ToBuilder() *EntityBuilder[ID] {
    if e == nil {
        return &EntityBuilder[ID]{}
    }
    var (
        Model_ID        ID
        Model_CreatedAt int64
        Model_UpdatedAt int64
    )
    if m := e.Model; m != nil {
        Model_ID = m.ID
        Model_CreatedAt = m.CreatedAt
        Model_UpdatedAt = m.UpdatedAt
    }

    return &EntityBuilder[ID]{
        iD:        Model_ID,
        createdAt: Model_CreatedAt,
        updatedAt: Model_UpdatedAt,
        name:      e.Name,
    }
}
```

## As-map usage example

source `struct.go`

``` go
package asmap

import "time"

//go:generate fieldr -type EmbeddedAddress -out address_as_map.go as-map -key-type . -export
//go:generate fieldr -type Struct -out struct_as_map.go as-map -key-type . -export -rewrite type:*EmbeddedAddress:fmt=%v.AsMap() -flat Flat

type BaseStruct struct {
    ID int
    TS *time.Time
}

type EmbeddedAddress struct {
    ZipCode     int
    AddressLine string
}

type FlatPart struct {
    CardNum string
    Bank    string
}

type Struct[n string] struct {
    *BaseStruct
    Name     n
    Surname  string
    noExport string //nolint
    NoTag    string
    Address  *EmbeddedAddress
    Flat     FlatPart
}
```

``` console
go generate .
```

will be generate two files `struct_as_map.go`, `address_as_map.go`

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package asmap

type StructField string

const (
    BaseStructID StructField = "ID"
    BaseStructTS StructField = "TS"
    Name         StructField = "Name"
    Surname      StructField = "Surname"
    NoTag        StructField = "NoTag"
    Address      StructField = "Address"
    FlatCardNum  StructField = "CardNum"
    FlatBank     StructField = "Bank"
)

func (s *Struct[n]) AsMap() map[StructField]interface{} {
    if s == nil {
        return nil
    }
    m := map[StructField]interface{}{}
    if bs := s.BaseStruct; bs != nil {
        m[BaseStructID] = bs.ID
    }
    if bs := s.BaseStruct; bs != nil {
        if ts := bs.TS; ts != nil {
            m[BaseStructTS] = ts
        }
    }
    m[Name] = s.Name
    m[Surname] = s.Surname
    m[NoTag] = s.NoTag
    if a := s.Address; a != nil {
        m[Address] = a.AsMap()
    }
    m[FlatCardNum] = s.Flat.CardNum
    m[FlatBank] = s.Flat.Bank
    return m
}
```

``` go
// Code generated by 'fieldr'; DO NOT EDIT.

package asmap

type EmbeddedAddressField string

const (
    ZipCode     EmbeddedAddressField = "ZipCode"
    AddressLine EmbeddedAddressField = "AddressLine"
)

func (e *EmbeddedAddress) AsMap() map[EmbeddedAddressField]interface{} {
    if e == nil {
        return nil
    }
    m := map[EmbeddedAddressField]interface{}{}
    m[ZipCode] = e.ZipCode
    m[AddressLine] = e.AddressLine
    return m
}
```

See more examples [here](./examples/)
