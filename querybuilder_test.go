package querybuilder

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestBuildDataHelperSelect(t *testing.T) {
	q := NewQueryBuilder("{TableNotSoImportant}")
	q.ResultLimitPosition = REAR
	q.InterpolateTables = true
	q.Schema = "carr"
	q.ResultLimit = "100"
	q.CommandType = SELECT
	q.AddColumn("UserKey").AddColumn("UserName").AddColumn("FullName").AddColumn("Gender").AddColumn("Age") //Method chaining
	q.AddFilter("Gender=1")
	q.AddOrder("Gender", ASC)
	q.AddGroup("FullName")

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		fmt.Println(vi)
	}
}

func TestBuildDataHelperInsert(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")

	q.PreparedStatementChar = "@p"
	q.PreparedStatementInSequence = true
	q.SkipNilWriteColumn = true //Ship if the value is null. Works only on INSERT and UPDATE

	q.CommandType = INSERT
	q.AddValue("UserKey", 5, nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Gender", nil, nil)
	q.AddValue("Birthdate", "GETDATE()", &ValueOption{false, nil, nil})

	var vbool interface{}
	vbool = true

	var pinf1 *interface{}
	pinf1 = &vbool

	var pinf2 interface{}
	pinf2 = false

	q.AddValue("PackedInterface1", pinf1, nil)
	q.AddValue("PackedInterface2", pinf2, nil)

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		fmt.Println(vi, vt.String())
	}
}

func TestBuildDataHelperUpdate(t *testing.T) {
	q := NewQueryBuilder("{TableNotSoImportant}")
	q.CommandType = UPDATE
	q.Schema = "hack"

	q.InterpolateTables = true

	q.PreparedStatementChar = "$"
	q.PreparedStatementInSequence = true
	q.SkipNilWriteColumn = true

	q.AddValue("UserKey", 5, nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Birthdate", "GETDATE()", &ValueOption{false, nil, nil})
	q.AddValue("DateCreated", "1/1/1900", &ValueOption{false, nil, "1/1/1900"})
	q.AddValue("Gender", nil, nil)
	q.AddFilter("CountryCode='PHL'")
	q.AddFilterWithValue("Town", "Manila", true)
	q.AddFilterWithValue("District", nil, true)

	/* The following commands are added to demonstrate that UPDATE command type does not support it, will panic. Uncomment to test*/
	//q.AddOrder("UserName", ASC)
	//q.AddGroup("UserName")

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		if vi != nil {
			vt := reflect.TypeOf(vi)
			fmt.Println(vi, vt.String())
		}
	}
}

func TestBuildDataHelperDelete(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")

	q.PreparedStatementChar = "$"
	q.PreparedStatementInSequence = true

	q.CommandType = DELETE
	q.AddFilter("CountryCode='PHL'")
	q.AddFilterWithValue("Town", "Manila", true)
	q.AddFilterWithValue("District", nil, true)

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

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		fmt.Println(vi, vt.String())
	}
}

func TestBuildStringSelect(t *testing.T) {
	q := NewQueryBuilder("[TableNotSoImportant]")

	q.ResultLimitPosition = REAR
	q.ResultLimit = "100"
	q.CommandType = SELECT
	q.AddColumn("UserKey").AddColumn("UserName").AddColumn("FullName").AddColumn("Gender").AddColumn("Age")
	q.AddFilter("Gender=1")
	q.AddOrder("Gender", ASC)
	q.AddGroup("FullName")

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestBuildStringInsert(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.SkipNilWriteColumn = true

	q.CommandType = INSERT
	q.AddValue("UserKey", float64(5), nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Birthdate", time.Now(), nil)
	q.AddValue("Town", nil, nil)
	q.AddValue("DateLastLoggedIn", "GETDATE()", &ValueOption{false, nil, nil})             /* Useful for calling SQL functions*/
	q.AddValue("ProfileImageURL", nil, &ValueOption{true, "http://www.facebook.com", nil}) /* Autodetects nil value and replaces the default value*/
	q.AddValue("ImageURL", "about:config", &ValueOption{true, nil, "about:config"})        /* When the null detect value matches the value, it sets the value to null */

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestBuildStringUpdate(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.SkipNilWriteColumn = false

	q.CommandType = UPDATE
	q.AddValue("UserKey", float64(5), nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Birthdate", time.Now(), nil)
	q.AddValue("Town", nil, nil)
	q.AddValue("DateLastLoggedIn", "GETDATE()", &ValueOption{false, nil, nil})             /* Useful for calling SQL functions*/
	q.AddValue("ProfileImageURL", nil, &ValueOption{true, "http://www.facebook.com", nil}) /* Autodetects nil value and replaces the default value*/

	q.AddFilter("Active = 1")
	q.AddFilterWithValue("ActivationStatus", "ACTIVATED", true)
	q.AddFilterWithValue("ActivationCode", "UNIQUEID()", false) /* Useful for calling SQL functions for filter */
	q.AddFilterWithValue("ActivationStatus", nil, true)

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestBuildStringDelete(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.CommandType = DELETE

	/* The following commands are added to demonstrate that DELETE command ignores it */
	q.AddValue("UserKey", float64(5), nil)
	q.AddValue("UserName", "eaglebush", nil)
	q.AddValue("Alias", "zaldy.baguinon", nil)
	q.AddValue("FullName", "Elizalde Baguinon", nil)
	q.AddValue("Active", false, nil)
	q.AddValue("Birthdate", time.Now(), nil)
	q.AddValue("DateLastLoggedIn", "GETDATE()", &ValueOption{false, nil, nil})             /* Useful for calling SQL functions*/
	q.AddValue("ProfileImageURL", nil, &ValueOption{true, "http://www.facebook.com", nil}) /* Autodetects nil value and replaces the default value*/

	q.AddFilter("Active = 1") /* For a non-equal filter */
	q.AddFilterWithValue("ActivationStatus", "ACTIVATED", true)
	q.AddFilterWithValue("ActivationCode", "UNIQUEID()", false) /* Useful for calling SQL functions for filter */
	q.AddFilterWithValue("ActivationStatus", nil, true)

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestTimeFormat(t *testing.T) {
	tm := time.Now()
	fmt.Printf("Time :%v", tm.Format(`2006-01-02 15:04:05`))
}
