-- For the attached database
create table users (
	id int primary key not null
);

create table sponsors (
	id int primary key not null
);

create table videos (
	id int primary key not null,

	user_id int not null,
	sponsor_id int unique,

	foreign key (user_id) references users (id),
	foreign key (sponsor_id) references sponsors (id)
);

create table tags (
	id int primary key not null
);

create table video_tags (
	video_id int not null,
	tag_id   int not null,

	primary key (video_id, tag_id),
	foreign key (video_id) references videos (id),
	foreign key (tag_id) references tags (id)
);


-- all table defintions will not cause sqlite autoincrement primary key without rowid tables to be generated
create table autoinctest (
	id INTEGER PRIMARY KEY NOT NULL
);

-- additional fields should not be marked as auto generated, when the AUTOINCREMENT keyword is present
create table autoinckeywordtest (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	b INTEGER
);

create view user_videos as
select u.id user_id, v.id video_id, v.sponsor_id sponsor_id
from users u
inner join videos v on v.user_id = u.id;

create table as_generated_columns (
   a INTEGER PRIMARY KEY NOT NULL,
   b INT,
   c TEXT,
   d INT GENERATED ALWAYS AS (a*abs(b)) VIRTUAL,
   e TEXT GENERATED ALWAYS AS (substr(c,b,b+1)) STORED
);

CREATE TABLE foo_bar (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_col TEXT NOT NULL
);
CREATE TABLE foo_baz (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_col TEXT NOT NULL
);
CREATE TABLE foo_qux (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_col TEXT NOT NULL
);
CREATE TABLE bar_baz (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_col TEXT NOT NULL
);
CREATE TABLE bar_qux (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_col TEXT NOT NULL
);
