package hook

import (
	"context"
	"log"
	"time"

	"github.com/mikestefanello/pagoda/ent"
)

// EnsureUTCHook creates a hook that ensures specified time fields are in UTC.
func EnsureUTCHook(timeFields ...string) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			// Iterate over provided time field names and adjust them to UTC
			for _, fieldName := range timeFields {
				adjustTimeFieldToUTC(m, fieldName)
			}
			return next.Mutate(ctx, m)
		})
	}
}

// adjustTimeFieldToUTC checks if a time field is set in the mutation and adjusts it to UTC.
func adjustTimeFieldToUTC(m ent.Mutation, fieldName string) {
	if value, exists := m.Field(fieldName); exists {
		if t, ok := value.(time.Time); ok {
			// Ensure the time is in UTC before setting it back
			m.SetField(fieldName, t.UTC())
		} else {
			log.Printf("Field %s is not a time.Time type", fieldName)
		}
	}
}
