package helper

import (
  "fmt"
  "log"
  "reflect"
  "strconv"
  "strings"
)

func Vector(array interface{}) string {
  sliceValue := reflect.ValueOf(array)

  if sliceValue.Kind() != reflect.Slice {
    return "Not a slice"
  }

  strSlice := make([]string, sliceValue.Len())
  for i := 0; i < sliceValue.Len(); i++ {
    element := sliceValue.Index(i)
    strSlice[i] = fmt.Sprintf("%v", element.Interface())
  }

  result := "[" + strings.Join(strSlice, ", ") + "]"
  return result
}

func VectorToString(vector []float32) string {
  var sb strings.Builder
  sb.WriteString("{")
  for i, v := range vector {
    if i > 0 {
      sb.WriteString(",")
    }
    sb.WriteString(fmt.Sprintf("%f", v))
  }
  sb.WriteString("}")
  return sb.String()
}

func StringToVector(s string) ([]float64, error) {
  s = strings.Trim(s, "{}")
  s = strings.Trim(s, "[]")
  parts := strings.Split(s, ",")
  vector := make([]float64, len(parts))
  for i, p := range parts {
    var err error
    vector[i], err = strconv.ParseFloat(strings.TrimSpace(p), 64)
    if err != nil {
      log.Println(err)
      return nil, err
    }
  }
  return vector, nil
}
