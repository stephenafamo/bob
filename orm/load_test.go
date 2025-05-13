package orm

// Both Loader and Preloader should be usable as a PreloadOption
var (
	_ PreloadOption[Loadable] = Loader[Loadable](nil)
	_ PreloadOption[Loadable] = Preloader[Loadable](nil)
)
