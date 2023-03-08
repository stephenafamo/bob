table "sponsors" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  primary_key {
    columns = [column.id]
  }
}
table "tags" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  primary_key {
    columns = [column.id]
  }
}
table "type_monsters" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "enum_use" {
    null = false
    type = enum.workday
  }
  column "enum_nullable" {
    null = true
    type = enum.workday
  }
  column "bool_zero" {
    null = true
    type = boolean
  }
  column "bool_one" {
    null = true
    type = boolean
  }
  column "bool_two" {
    null = false
    type = boolean
  }
  column "bool_three" {
    null    = true
    type    = boolean
    default = false
  }
  column "bool_four" {
    null    = true
    type    = boolean
    default = true
  }
  column "bool_five" {
    null    = false
    type    = boolean
    default = false
  }
  column "bool_six" {
    null    = false
    type    = boolean
    default = true
  }
  column "string_zero" {
    null = true
    type = character_varying(1)
  }
  column "string_one" {
    null = true
    type = character_varying(1)
  }
  column "string_two" {
    null = false
    type = character_varying(1)
  }
  column "string_three" {
    null    = true
    type    = character_varying(1)
    default = "a"
  }
  column "string_four" {
    null    = false
    type    = character_varying(1)
    default = "b"
  }
  column "string_five" {
    null = true
    type = character_varying(1000)
  }
  column "string_six" {
    null = true
    type = character_varying(1000)
  }
  column "string_seven" {
    null = false
    type = character_varying(1000)
  }
  column "string_eight" {
    null    = true
    type    = character_varying(1000)
    default = "abcdefgh"
  }
  column "string_nine" {
    null    = false
    type    = character_varying(1000)
    default = "abcdefgh"
  }
  column "string_ten" {
    null    = true
    type    = character_varying(1000)
    default = ""
  }
  column "string_eleven" {
    null    = false
    type    = character_varying(1000)
    default = ""
  }
  column "nonbyte_zero" {
    null = true
    type = character(1)
  }
  column "nonbyte_one" {
    null = true
    type = character(1)
  }
  column "nonbyte_two" {
    null = false
    type = character(1)
  }
  column "nonbyte_three" {
    null    = true
    type    = character(1)
    default = "a"
  }
  column "nonbyte_four" {
    null    = false
    type    = character(1)
    default = "b"
  }
  column "nonbyte_five" {
    null = true
    type = character(1000)
  }
  column "nonbyte_six" {
    null = true
    type = character(1000)
  }
  column "nonbyte_seven" {
    null = false
    type = character(1000)
  }
  column "nonbyte_eight" {
    null    = true
    type    = character(1000)
    default = "a"
  }
  column "nonbyte_nine" {
    null    = false
    type    = character(1000)
    default = "b"
  }
  column "byte_zero" {
    null = true
    type = sql("\"char\"")
  }
  column "byte_one" {
    null = true
    type = sql("\"char\"")
  }
  column "byte_two" {
    null    = true
    type    = sql("\"char\"")
    default = sql("'a'::\"char\"")
  }
  column "byte_three" {
    null = false
    type = sql("\"char\"")
  }
  column "byte_four" {
    null    = false
    type    = sql("\"char\"")
    default = sql("'b'::\"char\"")
  }
  column "big_int_zero" {
    null = true
    type = bigint
  }
  column "big_int_one" {
    null = true
    type = bigint
  }
  column "big_int_two" {
    null = false
    type = bigint
  }
  column "big_int_three" {
    null    = true
    type    = bigint
    default = 111111
  }
  column "big_int_four" {
    null    = false
    type    = bigint
    default = 222222
  }
  column "big_int_five" {
    null    = true
    type    = bigint
    default = 0
  }
  column "big_int_six" {
    null    = false
    type    = bigint
    default = 0
  }
  column "int_zero" {
    null = true
    type = integer
  }
  column "int_one" {
    null = true
    type = integer
  }
  column "int_two" {
    null = false
    type = integer
  }
  column "int_three" {
    null    = true
    type    = integer
    default = 333333
  }
  column "int_four" {
    null    = false
    type    = integer
    default = 444444
  }
  column "int_five" {
    null    = true
    type    = integer
    default = 0
  }
  column "int_six" {
    null    = false
    type    = integer
    default = 0
  }
  column "float_zero" {
    null = true
    type = numeric
  }
  column "float_one" {
    null = true
    type = numeric
  }
  column "float_two" {
    null = true
    type = numeric(2,1)
  }
  column "float_three" {
    null = true
    type = numeric(2,1)
  }
  column "float_four" {
    null = true
    type = numeric(2,1)
  }
  column "float_five" {
    null = false
    type = numeric(2,1)
  }
  column "float_six" {
    null    = true
    type    = numeric(2,1)
    default = 1.1
  }
  column "float_seven" {
    null    = false
    type    = numeric(2,1)
    default = 1.1
  }
  column "float_eight" {
    null    = true
    type    = numeric(2,1)
    default = 0
  }
  column "float_nine" {
    null    = true
    type    = numeric(2,1)
    default = 0
  }
  column "bytea_zero" {
    null = true
    type = bytea
  }
  column "bytea_one" {
    null = true
    type = bytea
  }
  column "bytea_two" {
    null = false
    type = bytea
  }
  column "bytea_three" {
    null    = false
    type    = bytea
    default = "\\x61"
  }
  column "bytea_four" {
    null    = true
    type    = bytea
    default = "\\x62"
  }
  column "bytea_five" {
    null    = false
    type    = bytea
    default = "\\x616263646566676861626364656667686162636465666768"
  }
  column "bytea_six" {
    null    = true
    type    = bytea
    default = "\\x686766656463626168676665646362616867666564636261"
  }
  column "bytea_seven" {
    null    = false
    type    = bytea
    default = "\\x"
  }
  column "bytea_eight" {
    null    = false
    type    = bytea
    default = "\\x"
  }
  column "time_zero" {
    null = true
    type = timestamp
  }
  column "time_one" {
    null = true
    type = date
  }
  column "time_two" {
    null = true
    type = timestamp
  }
  column "time_three" {
    null = true
    type = timestamp
  }
  column "time_four" {
    null = false
    type = timestamp
  }
  column "time_five" {
    null    = true
    type    = timestamp
    default = "1999-01-08 04:05:06.789"
  }
  column "time_six" {
    null    = true
    type    = timestamp
    default = "1999-01-08 04:05:06.789"
  }
  column "time_seven" {
    null    = true
    type    = timestamp
    default = "1999-01-08 04:05:06"
  }
  column "time_eight" {
    null    = false
    type    = timestamp
    default = "1999-01-08 04:05:06.789"
  }
  column "time_nine" {
    null    = false
    type    = timestamp
    default = "1999-01-08 04:05:06.789"
  }
  column "time_ten" {
    null    = false
    type    = timestamp
    default = "1999-01-08 04:05:06"
  }
  column "time_eleven" {
    null = true
    type = date
  }
  column "time_twelve" {
    null = false
    type = date
  }
  column "time_thirteen" {
    null    = true
    type    = date
    default = "1999-01-08"
  }
  column "time_fourteen" {
    null    = true
    type    = date
    default = "1999-01-08"
  }
  column "time_fifteen" {
    null    = true
    type    = date
    default = "1999-01-08"
  }
  column "time_sixteen" {
    null    = false
    type    = date
    default = "1999-01-08"
  }
  column "time_seventeen" {
    null    = false
    type    = date
    default = "1999-01-08"
  }
  column "time_eighteen" {
    null    = false
    type    = date
    default = "1999-01-08"
  }
  column "uuid_zero" {
    null = true
    type = uuid
  }
  column "uuid_one" {
    null = true
    type = uuid
  }
  column "uuid_two" {
    null = true
    type = uuid
  }
  column "uuid_three" {
    null = false
    type = uuid
  }
  column "uuid_four" {
    null    = true
    type    = uuid
    default = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }
  column "uuid_five" {
    null    = false
    type    = uuid
    default = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }
  column "integer_default" {
    null    = true
    type    = integer
    default = 5
  }
  column "varchar_default" {
    null    = true
    type    = character_varying(1000)
    default = sql("(5)::character varying")
  }
  column "timestamp_notz" {
    null    = true
    type    = timestamp
    default = sql("(now() AT TIME ZONE 'utc'::text)")
  }
  column "timestamp_tz" {
    null    = true
    type    = timestamptz
    default = sql("(now() AT TIME ZONE 'utc'::text)")
  }
  column "interval_nnull" {
    null    = false
    type    = interval
    default = sql("'21 days'::interval")
  }
  column "interval_null" {
    null    = true
    type    = interval
    default = sql("'23:00:00'::interval")
  }
  column "json_null" {
    null = true
    type = json
  }
  column "json_nnull" {
    null = false
    type = json
  }
  column "jsonb_null" {
    null = true
    type = jsonb
  }
  column "jsonb_nnull" {
    null = false
    type = jsonb
  }
  column "box_null" {
    null = true
    type = box
  }
  column "box_nnull" {
    null = false
    type = box
  }
  column "cidr_null" {
    null = true
    type = cidr
  }
  column "cidr_nnull" {
    null = false
    type = cidr
  }
  column "circle_null" {
    null = true
    type = circle
  }
  column "circle_nnull" {
    null = false
    type = circle
  }
  column "double_prec_null" {
    null = true
    type = double_precision
  }
  column "double_prec_nnull" {
    null = false
    type = double_precision
  }
  column "inet_null" {
    null = true
    type = inet
  }
  column "inet_nnull" {
    null = false
    type = inet
  }
  column "line_null" {
    null = true
    type = line
  }
  column "line_nnull" {
    null = false
    type = line
  }
  column "lseg_null" {
    null = true
    type = lseg
  }
  column "lseg_nnull" {
    null = false
    type = lseg
  }
  column "macaddr_null" {
    null = true
    type = macaddr
  }
  column "macaddr_nnull" {
    null = false
    type = macaddr
  }
  column "money_null" {
    null = true
    type = money
  }
  column "money_nnull" {
    null = false
    type = money
  }
  column "path_null" {
    null = true
    type = path
  }
  column "path_nnull" {
    null = false
    type = path
  }
  column "pg_lsn_null" {
    null = true
    type = sql("pg_lsn")
  }
  column "pg_lsn_nnull" {
    null = false
    type = sql("pg_lsn")
  }
  column "point_null" {
    null = true
    type = point
  }
  column "point_nnull" {
    null = false
    type = point
  }
  column "polygon_null" {
    null = true
    type = polygon
  }
  column "polygon_nnull" {
    null = false
    type = polygon
  }
  column "tsquery_null" {
    null = true
    type = tsquery
  }
  column "tsquery_nnull" {
    null = false
    type = tsquery
  }
  column "tsvector_null" {
    null = true
    type = tsvector
  }
  column "tsvector_nnull" {
    null = false
    type = tsvector
  }
  column "txid_null" {
    null = true
    type = sql("txid_snapshot")
  }
  column "txid_nnull" {
    null = false
    type = sql("txid_snapshot")
  }
  column "xml_null" {
    null = true
    type = xml
  }
  column "xml_nnull" {
    null = false
    type = xml
  }
  column "intarr_null" {
    null = true
    type = sql("integer[]")
  }
  column "intarr_nnull" {
    null = false
    type = sql("integer[]")
  }
  column "boolarr_null" {
    null = true
    type = sql("boolean[]")
  }
  column "boolarr_nnull" {
    null = false
    type = sql("boolean[]")
  }
  column "varchararr_null" {
    null = true
    type = sql("character varying[]")
  }
  column "varchararr_nnull" {
    null = false
    type = sql("character varying[]")
  }
  column "decimalarr_null" {
    null = true
    type = sql("numeric[]")
  }
  column "decimalarr_nnull" {
    null = false
    type = sql("numeric[]")
  }
  column "byteaarr_null" {
    null = true
    type = sql("bytea[]")
  }
  column "byteaarr_nnull" {
    null = false
    type = sql("bytea[]")
  }
  column "jsonbarr_null" {
    null = true
    type = sql("jsonb[]")
  }
  column "jsonbarr_nnull" {
    null = false
    type = sql("jsonb[]")
  }
  column "jsonarr_null" {
    null = true
    type = sql("json[]")
  }
  column "jsonarr_nnull" {
    null = false
    type = sql("json[]")
  }
  column "enumarr_null" {
    null = true
    type = sql("workday[]")
  }
  column "enumarr_nnull" {
    null = false
    type = sql("workday[]")
  }
  column "customarr_null" {
    null = true
    type = sql("my_int_array")
  }
  column "customarr_nnull" {
    null = false
    type = sql("my_int_array")
  }
  column "domainuint3_nnull" {
    null = false
    type = sql("uint3")
  }
  column "base" {
    null = true
    type = text
  }
  column "generated_nnull" {
    null = false
    type = text
    as {
      expr = "upper(base)"
      type = STORED
    }
  }
  column "generated_null" {
    null = true
    type = text
    as {
      expr = "upper(base)"
      type = STORED
    }
  }
  primary_key {
    columns = [column.id]
  }
}
table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "email_validated" {
    null    = true
    type    = boolean
    default = false
    comment = "Has the email address been tested?"
  }
  column "primary_email" {
    null    = true
    type    = character_varying(100)
    comment = "The user's preferred email address.\n\nUse this to send emails to the user."
  }
  primary_key {
    columns = [column.id]
  }
  index "users_primary_email_key" {
    unique  = true
    columns = [column.primary_email]
  }
}
table "video_tags" {
  schema = schema.public
  column "video_id" {
    null = false
    type = integer
  }
  column "tag_id" {
    null = false
    type = integer
  }
  primary_key {
    columns = [column.video_id, column.tag_id]
  }
  foreign_key "video_tags_tag_id_fkey" {
    columns     = [column.tag_id]
    ref_columns = [table.tags.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "video_tags_video_id_fkey" {
    columns     = [column.video_id]
    ref_columns = [table.videos.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
table "videos" {
  schema = schema.public
  column "id" {
    null    = false
    type    = serial
    comment = "The ID of the video"
  }
  column "user_id" {
    null = false
    type = integer
  }
  column "sponsor_id" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "videos_sponsor_id_fkey" {
    columns     = [column.sponsor_id]
    ref_columns = [table.sponsors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "videos_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "videos_sponsor_id_key" {
    unique  = true
    columns = [column.sponsor_id]
  }
}
enum "workday" {
  schema = schema.public
  values = ["monday", "tuesday", "wednesday", "thursday", "friday"]
}
schema "public" {
}
