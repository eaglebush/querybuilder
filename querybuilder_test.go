package querybuilder

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestBuildDataHelperSelect(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.ResultLimitPosition = REAR
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
	q.CommandType = INSERT
	q.AddColumnValue("UserKey", 5)
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnNonStringValue("Birthdate", "GETDATE()")

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		fmt.Println(vi, vt.String())
	}
}

func TestBuildDataHelperUpdate(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.CommandType = UPDATE
	q.AddColumnValue("UserKey", 5)
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnNonStringValue("Birthdate", "GETDATE()")
	q.AddColumnValueNull("DateCreated", "1/1/1900", "1/1/1900")
	q.AddFilter("CountryCode='PHL'")
	q.AddFilterWithValue("Town", "Manila")

	/* The following commands are added to demonstrate that UPDATE command type does not support it, will panic. Uncomment to test*/
	//q.AddOrder("UserName", ASC)
	//q.AddGroup("UserName")

	s, v := q.BuildDataHelper()
	fmt.Println(s)
	for _, vi := range v {
		vt := reflect.TypeOf(vi)
		fmt.Println(vi, vt.String())
	}
}

func TestBuildDataHelperDelete(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.CommandType = DELETE
	q.AddFilter("CountryCode='PHL'")
	q.AddFilterWithValue("Town", "Manila")

	/* The following commands are added to demonstrate that DELETE command ignores it */
	q.AddColumnValue("UserKey", 5)
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnValue("Birthdate", time.Now())

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
	q.CommandType = INSERT
	q.AddColumnValue("UserKey", float64(5))
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnValue("Birthdate", time.Now())
	q.AddColumnNonStringValue("DateLastLoggedIn", "GETDATE()")                     /* Useful for calling SQL functions*/
	q.AddColumnValueWithDefault("ProfileImageURL", nil, "http://www.facebook.com") /* Autodetects nil value and replaces the default value*/

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestBuildStringUpdate(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.CommandType = UPDATE
	q.AddColumnValue("UserKey", float64(5))
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnValue("Birthdate", time.Now())
	q.AddColumnNonStringValue("DateLastLoggedIn", "GETDATE()")                     /* Useful for calling SQL functions*/
	q.AddColumnValueWithDefault("ProfileImageURL", nil, "http://www.facebook.com") /* Autodetects nil value and replaces the default value*/

	q.AddFilter("Active = 1")
	q.AddFilterWithValue("ActivationStatus", "ACTIVATED")
	q.AddFilterWithNonStringValue("ActivationCode", "UNIQUEID()") /* Useful for calling SQL functions for filter */

	s, _ := q.BuildString()
	fmt.Println(s)
}

func TestBuildStringDelete(t *testing.T) {
	q := NewQueryBuilder("TableNotSoImportant")
	q.CommandType = DELETE

	/* The following commands are added to demonstrate that DELETE command ignores it */
	q.AddColumnValue("UserKey", float64(5))
	q.AddColumnValue("UserName", "eaglebush")
	q.AddColumnValue("Alias", "zaldy.baguinon")
	q.AddColumnValue("FullName", "Elizalde Baguinon")
	q.AddColumnValue("Active", false)
	q.AddColumnValue("Birthdate", time.Now())
	q.AddColumnNonStringValue("DateLastLoggedIn", "GETDATE()")                     /* Useful for calling SQL functions*/
	q.AddColumnValueWithDefault("ProfileImageURL", nil, "http://www.facebook.com") /* Autodetects nil value and replaces the default value*/

	q.AddFilter("Active = 1") /* For a non-equal filter */
	q.AddFilterWithValue("ActivationStatus", "ACTIVATED")
	q.AddFilterWithNonStringValue("ActivationCode", "UNIQUEID()") /* Useful for calling SQL functions for filter */

	s, _ := q.BuildString()
	fmt.Println(s)
}
