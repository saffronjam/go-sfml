{{ define "go_sfml_vector.go.tpl" }}
{{- $TypeName := .TypeName -}}
{{- $Name := .Name -}}
{{- $Components := .Components -}}
{{- $HasFloat := .HasFloat -}}

// ------------------- {{$Name}} Methods -------------------

/*
Add returns a new vector representing the component-wise sum
of this vector and another.

Params:
  - other: the vector to add to this one.

Returns:
  - A new {{$Name}} representing the result of the addition.
*/
func (v *{{$Name}}) Add(other *{{$Name}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} + other.{{.}},
		{{- end }}
	}
}

/*
Subtract returns a new vector representing the component-wise difference
between this vector and another.

Params:
  - other: the vector to subtract from this one.

Returns:
  - A new {{$Name}} representing the result of the subtraction.
*/
func (v *{{$Name}}) Subtract(other *{{$Name}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} - other.{{.}},
		{{- end }}
	}
}

/*
Multiply returns a new vector with component-wise multiplication
of this vector and another.

Params:
  - other: the vector to multiply with.

Returns:
  - A new {{$Name}} with each component multiplied.
*/
func (v *{{$Name}}) Multiply(other *{{$Name}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} * other.{{.}},
		{{- end }}
	}
}

/*
MultiplyScalar multiplies each component of the vector by a scalar.

Params:
  - scalar: the scalar value to multiply by.

Returns:
  - A new {{$Name}} with each component scaled.
*/
func (v *{{$Name}}) MultiplyScalar(scalar {{$TypeName}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} * scalar,
		{{- end }}
	}
}

/*
MultiplyScalars performs component-wise multiplication with individual scalar values.

Params:
  - {{ range $i, $e := $Components }}{{ if $i }}, {{ end }}{{ $e | ToLower }} {{$TypeName}}{{ end }}

Returns:
  - A new {{$Name}} with each component multiplied by its corresponding scalar.
*/
func (v *{{$Name}}) MultiplyScalars({{ range $i, $e := $Components }}{{ if $i }}, {{ end }}{{ $e | ToLower }} {{ $TypeName }}{{ end }}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} * {{. | ToLower}},
		{{- end }}
	}
}

/*
Divide performs component-wise division of this vector by another.

Params:
  - other: the vector to divide by.

Returns:
  - A new {{$Name}} with each component divided.
*/
func (v *{{$Name}}) Divide(other *{{$Name}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} / other.{{.}},
		{{- end }}
	}
}

/*
DivideScalar divides each component by a scalar.

Params:
  - scalar: the scalar divisor.

Returns:
  - A new {{$Name}} with each component divided by scalar.
*/
func (v *{{$Name}}) DivideScalar(scalar {{$TypeName}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} / scalar,
		{{- end }}
	}
}

/*
DivideScalars divides each component by its corresponding scalar.

Params:
  - {{ range $i, $e := $Components }}{{ if $i }}, {{ end }}{{ $e | ToLower }} {{$TypeName}}{{ end }}

Returns:
  - A new {{$Name}} with each component divided.
*/
func (v *{{$Name}}) DivideScalars({{ range $i, $e := $Components }}{{ if $i }}, {{ end }}{{ $e | ToLower }} {{ $TypeName }}{{ end }}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} / {{. | ToLower}},
		{{- end }}
	}
}

/*
Equals returns true if all components of both vectors are equal.

Params:
  - other: the vector to compare with.

Returns:
  - Boolean indicating equality.
*/
func (v *{{$Name}}) Equals(other *{{$Name}}) bool {
	return {{ range $i, $e := $Components }}{{ if $i }} && {{ end }}v.{{.}} == other.{{.}}{{ end }}
}

/*
String returns a formatted string representation of the vector.
*/
func (v *{{$Name}}) String() string {
	{{ if $HasFloat -}}
	return fmt.Sprintf("{{$Name}}({{- range $i, $e := $Components }}{{ if $i }}, {{ end }}{{.}}: %f{{ end }})", {{ range $i, $e := $Components }}{{ if $i }}, {{ end }}v.{{.}}{{ end }})
	{{- else -}}
	return fmt.Sprintf("{{$Name}}({{- range $i, $e := $Components }}{{ if $i }}, {{ end }}{{.}}: %d{{ end }})", {{ range $i, $e := $Components }}{{ if $i }}, {{ end }}v.{{.}}{{ end }})
	{{- end }}
}

{{ if $HasFloat }}
// --- Float-Specific {{$Name}} Methods ---

/*
LengthSquared returns the squared magnitude of the vector.

Returns:
  - Sum of squares of components.
*/
func (v *{{$Name}}) LengthSquared() {{$TypeName}} {
	return {{ range $i, $e := $Components }}{{ if $i }} + {{ end }}v.{{.}}*v.{{.}}{{ end }}
}

/*
Length returns the Euclidean length (magnitude) of the vector.

Returns:
  - Square root of LengthSquared.
*/
func (v *{{$Name}}) Length() {{$TypeName}} {
	return {{$TypeName}}(math.Sqrt(float64(v.LengthSquared())))
}

/*
Normalize returns a unit vector pointing in the same direction.

Returns:
  - A normalized vector, or zero vector if original length is 0.
*/
func (v *{{$Name}}) Normalize() *{{$Name}} {
	if l := v.Length(); l != 0 {
		return v.DivideScalar(l)
	}
	return &{{$Name}}{}
}

/*
Dot returns the dot product with another vector.

Params:
  - other: the vector to dot with.

Returns:
  - Dot product (scalar).
*/
func (v *{{$Name}}) Dot(other *{{$Name}}) {{$TypeName}} {
	return {{ range $i, $e := $Components }}{{ if $i }} + {{ end }}v.{{.}}*other.{{.}}{{ end }}
}

/*
Distance returns the Euclidean distance between two vectors.

Params:
  - other: the vector to measure distance to.

Returns:
  - Distance as a float.
*/
func (v *{{$Name}}) Distance(other *{{$Name}}) {{$TypeName}} {
	return {{$TypeName}}(math.Sqrt(float64(v.DistanceSquared(other))))
}

/*
DistanceSquared returns the squared distance between two vectors.

Params:
  - other: the vector to measure distance to.

Returns:
  - Squared distance (faster if exact distance isn't needed).
*/
func (v *{{$Name}}) DistanceSquared(other *{{$Name}}) {{$TypeName}} {
	return {{ range $i, $e := $Components }}{{ if $i }} + {{ end }}(v.{{.}} - other.{{.}})*(v.{{.}} - other.{{.}}){{ end }}
}

/*
Lerp performs linear interpolation toward another vector.

Params:
  - other: target vector.
  - t: interpolation factor in [0, 1].

Returns:
  - Interpolated vector between this and other.
*/
func (v *{{$Name}}) Lerp(other *{{$Name}}, t {{$TypeName}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} + (other.{{.}} - v.{{.}})*t,
		{{- end }}
	}
}

/*
Clamp limits each component to the corresponding range.

Params:
  - min: minimum vector values.
  - max: maximum vector values.

Returns:
  - Clamped vector.
*/
func (v *{{$Name}}) Clamp(min, max *{{$Name}}) *{{$Name}} {
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: {{$TypeName}}(math.Max(float64(min.{{.}}), math.Min(float64(max.{{.}}), float64(v.{{.}})))),
		{{- end }}
	}
}

/*
Reflect reflects this vector around a surface normal.

Params:
  - normal: surface normal vector.

Returns:
  - Reflected vector.
*/
func (v *{{$Name}}) Reflect(normal *{{$Name}}) *{{$Name}} {
	dot := v.Dot(normal)
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: v.{{.}} - 2*dot*normal.{{.}},
		{{- end }}
	}
}

/*
Project projects this vector onto another.

Params:
  - other: vector to project onto.

Returns:
  - Projected vector.
*/
func (v *{{$Name}}) Project(other *{{$Name}}) *{{$Name}} {
	dot := v.Dot(other)
	lengthSquared := other.LengthSquared()
	if lengthSquared == 0 {
		return &{{$Name}}{}
	}
	scalar := dot / lengthSquared
	return &{{$Name}}{
		{{- range $Components }}
		{{.}}: other.{{.}} * scalar,
		{{- end }}
	}
}

/*
SetLength returns a new vector in the same direction with a given length.

Params:
  - length: the desired length of the new vector.

Returns:
  - Rescaled vector, or zero vector if original length is zero.
*/
func (v *{{$Name}}) SetLength(length {{$TypeName}}) *{{$Name}} {
	if v.Length() == 0 {
		return &{{$Name}}{}
	}
	return v.Normalize().MultiplyScalar(length)
}

{{ if eq (len $Components) 2 }}
/*
Rotate rotates a 2D vector by a given angle in degrees.

Params:
  - angle: angle to rotate in degrees.

Returns:
  - Rotated vector.
*/
func (v *{{$Name}}) Rotate(angle {{$TypeName}}) *{{$Name}} {
	radians := angle * (math.Pi / 180.0)
	cos := {{$TypeName}}(math.Cos(float64(radians)))
	sin := {{$TypeName}}(math.Sin(float64(radians)))
	return &{{$Name}}{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
	}
}
{{ end }}
{{- end }}
{{- end }}