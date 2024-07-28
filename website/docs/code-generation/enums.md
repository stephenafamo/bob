---

sidebar_position: 8
description: Generated code for defined Enums

---

# Enums

Given enums defined as:

```sql
CREATE TYPE workday AS ENUM('monday', 'tuesday', 'wednesday', 'thursday', 'friday');
```

The following code is generated:

```go
type Workday string

// Enum values for Workday
const (
	WorkdayMonday    Workday = "monday"
	WorkdayTuesday   Workday = "tuesday"
	WorkdayWednesday Workday = "wednesday"
	WorkdayThursday  Workday = "thursday"
	WorkdayFriday    Workday = "friday"
)

func AllWorkday() []Workday {
	return []Workday{
		WorkdayMonday,
		WorkdayTuesday,
		WorkdayWednesday,
		WorkdayThursday,
		WorkdayFriday,
	}
}
```

This type is then used directly in the model to help with type safety and auto-completion.
