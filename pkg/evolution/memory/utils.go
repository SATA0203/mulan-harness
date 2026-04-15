package memory

import (
"fmt"
"time"
)

func generateMemoryID() string {
return fmt.Sprintf("mem-%d", time.Now().UnixNano())
}
