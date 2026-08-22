package musicplayer

import "errors"

// Neden: Go'da sentinel error kullanmak, çağıran kodun errors.Is ile
// belirli hata tiplerini ayırt edebilmesini sağlar (örn. "şarkı bulunamadı"
// ile "geçersiz index" farklı şekillerde handle edilebilir).
var (
	ErrEmptySongID     = errors.New("song id boş olamaz")
	ErrSongNotFound    = errors.New("şarkı bulunamadı")
	ErrDuplicateSong   = errors.New("bu id'ye sahip şarkı zaten playlist'te var")
	ErrInvalidIndex    = errors.New("geçersiz index")
	ErrEmptyPlaylist   = errors.New("playlist boş")
	ErrInvalidPosition = errors.New("geçersiz pozisyon")
)
