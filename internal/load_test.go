package internal

// Both Loader and Preloader should be usable as a PreloadOption
var (
	_ PreloadOption[loadable] = Loader[loadable](nil)
	_ PreloadOption[loadable] = Preloader[loadable](nil)
)
