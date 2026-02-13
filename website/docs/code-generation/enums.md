---

sidebar_position: 8
description: Generated code for defined Enums

---

# Enums

Given enums defined as:

```sql
CREATE TYPE task_status AS ENUM('not_started', 'in_progress', 'on_hold', 'completed');
```

The following code is generated:

```go
type TaskStatus string

// Enum values for TaskStatus
const (
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusOnHold     TaskStatus = "on_hold"
	TaskStatusCompleted  TaskStatus = "completed"
)

func AllTaskStatus() []TaskStatus {
	return []TaskStatus{
		TaskStatusNotStarted,
		TaskStatusInProgress,
		TaskStatusOnHold,
		TaskStatusCompleted,
	}
}
```

This type is then used directly in the model to help with type safety and auto-completion.

## Enum Value Formatting

You can control the format of enum value identifiers using the `enum_format` configuration option:

### Title Case (Default)

With `enum_format: "title_case"` (or no configuration), enum values are formatted in PascalCase:

```go
const (
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusOnHold     TaskStatus = "on_hold"
	TaskStatusCompleted  TaskStatus = "completed"
)
```

### Screaming Snake Case

With `enum_format: "screaming_snake_case"`, enum values are formatted in SCREAMING_SNAKE_CASE:

```yaml
# bobgen.yaml
enum_format: "screaming_snake_case"
```

```go
const (
	TaskStatusNOT_STARTED TaskStatus = "not_started"
	TaskStatusIN_PROGRESS TaskStatus = "in_progress"
	TaskStatusON_HOLD     TaskStatus = "on_hold"
	TaskStatusCOMPLETED   TaskStatus = "completed"
)
```

