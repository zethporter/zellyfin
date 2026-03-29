package types

type TitleDetail int

const (
	TD_DiskTitle       TitleDetail = 2
	TD_ChapterCount    TitleDetail = 8
	TD_LengthInSeconds TitleDetail = 9
	TD_FileSizeGB      TitleDetail = 10
	TD_FileSizeBytes   TitleDetail = 11
	TD_FileName        TitleDetail = 12
	TD_AudioShortCode  TitleDetail = 28
	TD_AudioLongCode   TitleDetail = 29
)

type TitleEntry struct {
	TitleID     int
	TitleDetail TitleDetail
	TitleValue  string
}

// Structure of the Disk Title list
type TitleInfo struct {
	ID        int
	Name      string
	Duration  string
	SizeHuman string
	SizeBytes int64
	Chapters  int
}

type FetchingProgress struct {
	Pct    int
	Titles map[int]TitleInfo
}
