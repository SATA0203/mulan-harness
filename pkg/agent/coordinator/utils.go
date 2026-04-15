package coordinator

import (
"fmt"
"time"
)

func generateTaskID() string {
return fmt.Sprintf("task-%d", time.Now().UnixNano())
}
