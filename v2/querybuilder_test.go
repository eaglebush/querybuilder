package querybuilder

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	fb "github.com/eaglebush/filterbuilder"
)

func TestBuildDataHelperSelect(t *testing.T) {

	q := New(WithTableName("{TableNotSoImportant}"), WithCommand(SELECT))
	q.ResultLimitPosition = REAR
	q.InterpolateTables = true
	q.Schema = "carr"
	q.ResultLimit = "100"

	q.AddColumn("UserKey").AddColumn("UserName").AddColumn("FullName").AddColumn("Gender").AddColumn("Age") //Method chaining
	q.AddFilterExp("Gender = 1")
	q.AddFilter("Orientation", nil)
	q.AddOrder("Gender", ASC)
	q.AddGroup("FullName")

	s, v, err := q.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	fmt.Println(s)
	for _, vi := range v {
		t.Log(vi)
	}
}

func TestBuildDataHelperSelectWithFilterValues(t *testing.T) {

	type sample struct {
		DomainCode      *string
		ApplicationCode *string
		ModuleCode      *string
		Code            *string
		Modifier        *string
	}

	ac := `APPSHUB-AUTH`

	data := sample{
		ApplicationCode: &ac,
		DomainCode:      new(string),
	}

	*data.DomainCode = "VDI"

	qb := New(WithTableName("{TableNotSoImportant}"))

	qb.AddColumn(`record_key`)

	qb.AddFilter(`domain_code`, data.DomainCode)
	qb.AddFilter(`application_code`, data.ApplicationCode)
	qb.AddFilter(`module_code`, data.ModuleCode)
	qb.AddFilter(`code`, data.Code)
	qb.AddFilter(`modifier`, data.Modifier)
	qb.AddFilter(`wildcard`, nil)

	sql, values, err := qb.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	t.Logf("Query: %s, Values: %v", sql, values)
}

func TestBuildDataHelperSelectWithFilterBuilderValues(t *testing.T) {

	type sample struct {
		DomainCode      *string
		ApplicationCode *string
		ModuleCode      *string
		Code            *string
		Modifier        *string
	}

	type simpleData struct {
		FirstName *string
		LastName  *string
		Age       int
	}

	fn := "Zaldy"
	ln := "Baguinon"

	sd := simpleData{
		FirstName: &fn,
		LastName:  &ln,
		Age:       46,
	}

	n := fb.Null(true)

	fbv := fb.Filter{
		Data: sd,
		Eq: []fb.Pair{
			{Column: "first_name", Value: fb.Value{Src: "FirstName"}},
			{Column: "last_name", Value: fb.Value{Src: n, Raw: true}},
		},
		Ne: []fb.Pair{
			{Column: "first_name", Value: fb.Value{Src: "FirstName"}},
			{Column: "last_name", Value: fb.Value{Src: "LastName"}},
		},
		Lk: []fb.Pair{
			{Column: "first_name", Value: fb.Value{Src: "FirstName"}},
			{Column: "last_name", Value: fb.Value{Src: "LastName"}},
		},
	}

	_ = fbv

	ac := `APPSHUB-AUTH`

	data := sample{
		ApplicationCode: &ac,
		DomainCode:      new(string),
	}

	*data.DomainCode = "VDI"

	qb := New(WithTableName("{TableNotSoImportant}"))
	qb.ParameterInSequence = true
	qb.ParameterChar = "@p"

	qb.AddColumn(`record_key`)
	qb.AddFilter(`domain_code`, data.DomainCode)
	qb.AddFilter(`application_code`, data.ApplicationCode)
	qb.AddFilter(`module_code`, data.ModuleCode)
	qb.AddFilter(`code`, data.Code)
	qb.AddFilter(`modifier`, data.Modifier)
	qb.AddFilter(`wildcard`, nil)

	qb.AddFilterExp("Wacky = Yes")

	// qb.FilterFunc = func(paramoffset int, paramchar string, paraminseq bool) ([]string, []interface{}) {
	// 	fbv.ParameterOffset = paramoffset
	// 	fbv.ParameterPlaceholder = paramchar
	// 	fbv.ParameterInSequence = paraminseq
	// 	s, a, err := fbv.Build()
	// 	if err != nil {
	// 		log.Printf("error: %s", err)
	// 	}
	// 	return s, a
	// }

	// qb.FilterFunc = func(paramoffset int, paramchar string, paraminseq bool) ([]string, []interface{}) {
	// 	return fbv.BuildFunc(paramoffset, paramchar, paraminseq)
	// }

	qb.FilterFunc = fbv.BuildFunc
	sql, values, err := qb.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	t.Logf("Query: %s, Values: %v", sql, values)
}

func TestBuildDataHelperInsert(t *testing.T) {
	q := New(WithTableName("{TableNotSoImportant}"))

	q.ParameterChar = "@p"
	q.ParameterInSequence = true
	q.SkipNilWriteColumn = false //Ship if the value is null. Works only on INSERT and UPDATE

	q.CommandType = INSERT
	q.AddValue("UserKey", 5, nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Gender", nil, nil) // if SkipNilWriteColumn is true, this will be skipped, else this will be set to null
	q.AddValue("Birthdate", "GETDATE()", IsSqlString(false))

	var vbool interface{}
	vbool = true

	var pinf1 *interface{}
	pinf1 = &vbool

	var pinf2 interface{}
	pinf2 = false

	q.AddValue("PackedInterface1", pinf1, nil)
	q.AddValue("PackedInterface2", pinf2, nil)

	q.AddValue("TraderAddrClassKey", 0, MatchToNull(0))

	s, v, err := q.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	t.Log(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		t.Log(vi, vt.String())
	}
}

func TestBuildDataHelperUpdate(t *testing.T) {
	q := New(WithTableName("{TableNotSoImportant}"), WithCommand(UPDATE))
	q.Schema = "hack"
	q.InterpolateTables = true
	q.ParameterChar = "$"
	q.ParameterInSequence = true
	q.SkipNilWriteColumn = false

	q.AddValue("UserKey", 5)
	q.AddValue("UserName", "eaglebush")
	q.AddValue("Alias", "zaldy.baguinon")
	q.AddValue("FullName", "Elizalde Baguinon")
	q.AddValue("Active", false)
	q.AddValue("Birthdate", "GETDATE()", IsSqlString(false))
	q.AddValue("DateCreated", "1/1/1900", IsSqlString(false), MatchToNull("1/1/1900"))
	q.AddValue("Gender", nil)
	q.AddFilterExp("CountryCode='PHL'")
	q.AddFilter("Town", "Manila")
	q.AddFilter("District", nil)

	/* The following commands are added to demonstrate that UPDATE command type does not support it, will panic. Uncomment to test*/
	//q.AddOrder("UserName", ASC)
	//q.AddGroup("UserName")

	s, v, err := q.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	t.Log(s)
	for _, vi := range v {
		if vi != nil {
			vt := reflect.TypeOf(vi)
			t.Log(vi, vt.String())
		}
	}
}

func TestBuildDataHelperDelete(t *testing.T) {
	q := New(WithTableName("{TableNotSoImportant}"), WithCommand(DELETE))
	q.ParameterChar = "$"
	q.ParameterInSequence = true

	q.AddFilterExp("CountryCode='PHL'")
	q.AddFilter("Town", "Manila")
	q.AddFilter("District", nil)

	/* The following commands are added to demonstrate that DELETE command ignores it */
	q.AddValue("UserKey", 5, nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Birthdate", time.Now(), nil)

	/* The following commands are added to demonstrate that DELETE command type command ignores it. Uncomment to test*/
	//q.AddOrder("UserName", ASC)
	//q.AddGroup("UserName")

	s, v, err := q.Build()
	if err != nil {
		t.Logf("Error: %e", err)
		return
	}

	t.Log(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		t.Log(vi, vt.String())
	}
}

func TestTimeFormat(t *testing.T) {
	tm := time.Now()
	t.Logf("Time :%v", tm.Format(`2006-01-02 15:04:05`))
}

func TestVariablePlainInterface(t *testing.T) {

	var (
		ret interface{}
	)

	input := make([]interface{}, 18)

	var (
		str string
		i   int
		i32 int32
		i64 int64
		f32 float32
		f64 float64
		tm  time.Time
		b   bool
		ba  []byte
	)

	str = "string"
	i = 100
	i32 = 1000
	i64 = 10000
	f32 = 12345.54321
	f64 = 123456789.987654321
	tm = time.Now()
	b = true
	ba = []byte("This is a good test")

	input[0] = str
	input[1] = i
	input[2] = i32
	input[3] = i64
	input[4] = f32
	input[5] = f64
	input[6] = tm
	input[7] = b
	input[8] = ba

	input[9] = &str
	input[10] = &i
	input[11] = &i32
	input[12] = &i64
	input[13] = &f32
	input[14] = &f64
	input[15] = &tm
	input[16] = &b
	input[17] = &ba

	for ix := range input {
		ret = realValue(input[ix])
		t.Logf("Value: %v", ret)
	}
}

func TestVariablePointerToInterface(t *testing.T) {

	var ret interface{}
	input := make([]*interface{}, 18)

	var (
		str string
		i   int
		i32 int32
		i64 int64
		f32 float32
		f64 float64
		tm  time.Time
		b   bool
		ba  []byte
	)

	str = "string"
	i = 100
	i32 = 1000
	i64 = 10000
	f32 = 12345.54321
	f64 = 123456789.987654321
	tm = time.Now()
	b = true
	ba = []byte("This is a good test")

	input[0] = new(interface{})
	input[1] = new(interface{})
	input[2] = new(interface{})
	input[3] = new(interface{})
	input[4] = new(interface{})
	input[5] = new(interface{})
	input[6] = new(interface{})
	input[7] = new(interface{})
	input[8] = new(interface{})
	input[9] = new(interface{})
	input[10] = new(interface{})
	input[11] = new(interface{})
	input[12] = new(interface{})
	input[13] = new(interface{})
	input[14] = new(interface{})
	input[15] = new(interface{})
	input[16] = new(interface{})
	input[17] = new(interface{})

	*input[0] = str
	*input[1] = i
	*input[2] = i32
	*input[3] = i64
	*input[4] = f32
	*input[5] = f64
	*input[6] = tm
	*input[7] = b
	*input[8] = ba

	*input[9] = &str
	*input[10] = &i
	*input[11] = &i32
	*input[12] = &i64
	*input[13] = &f32
	*input[14] = &f64
	*input[15] = &tm
	*input[16] = &b
	*input[17] = &ba

	for ix := range input {
		ret = realValue(input[ix])
		t.Logf("Value: %v", ret)
	}
}

func TestVariablePlainInterfaceStruct(t *testing.T) {

	var (
		str string
		i   int
		i32 int32
		i64 int64
		f32 float32
		f64 float64
		tm  time.Time
		b   bool
		ba  []byte
	)

	type samplestruct struct {
		str string
		i   int
		i32 int32
		i64 int64
		f32 float32
		f64 float64
		tm  time.Time
		b   bool
		ba  []byte
	}

	str = "string"
	i = 100
	i32 = 1000
	i64 = 10000
	f32 = 12345.54321
	f64 = 123456789.987654321
	tm = time.Now()
	b = true
	ba = []byte("This is a good test")

	ss := &samplestruct{
		str: str,
		i:   i,
		i32: i32,
		i64: i64,
		f32: f32,
		f64: f64,
		tm:  tm,
		b:   b,
		ba:  ba,
	}

	t.Logf("str: %v", realValue(ss.str))
	t.Logf("i: %v", realValue(ss.i))
	t.Logf("i32: %v", realValue(ss.i32))
	t.Logf("i64: %v", realValue(ss.i64))
	t.Logf("f32: %v", realValue(ss.f32))
	t.Logf("f64: %v", realValue(ss.f64))
	t.Logf("tm: %v", realValue(ss.tm))
	t.Logf("b: %v", realValue(ss.b))
	t.Logf("ba: %v", realValue(ss.ba))

}

func TestVariablePointerToInterfaceStruct(t *testing.T) {

	var (
		str string
		i   int
		i32 int32
		i64 int64
		f32 float32
		f64 float64
		tm  time.Time
		b   bool
		ba  []byte
	)

	type samplestruct struct {
		str *string
		i   *int
		i32 *int32
		i64 *int64
		f32 *float32
		f64 *float64
		tm  *time.Time
		b   *bool
		ba  *[]byte
	}

	str = "string"
	i = 100
	i32 = 1000
	i64 = 10000
	f32 = 12345.54321
	f64 = 123456789.987654321
	tm = time.Now()
	b = true
	ba = []byte("This is a good test")

	ss := &samplestruct{
		str: &str,
		i:   &i,
		i32: &i32,
		i64: &i64,
		f32: &f32,
		f64: &f64,
		tm:  &tm,
		b:   &b,
		ba:  &ba,
	}

	t.Logf("str: %v", realValue(ss.str))
	t.Logf("i: %v", realValue(ss.i))
	t.Logf("i32: %v", realValue(ss.i32))
	t.Logf("i64: %v", realValue(ss.i64))
	t.Logf("f32: %v", realValue(ss.f32))
	t.Logf("f64: %v", realValue(ss.f64))
	t.Logf("tm: %v", realValue(ss.tm))
	t.Logf("b: %v", realValue(ss.b))
	t.Logf("ba: %v", realValue(ss.ba))
}
