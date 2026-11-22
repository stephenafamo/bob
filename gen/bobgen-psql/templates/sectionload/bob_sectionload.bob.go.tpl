{{$.Importer.Import "sectionload" sectionloadPkg}}

// GeneratorContractVersion pins the data contract this package was generated
// for. The array length below is negative, and therefore fails to compile,
// whenever it differs from sectionload.ContractVersion.
const GeneratorContractVersion = 1

var _ [1]struct{} = [1 - (sectionload.ContractVersion-GeneratorContractVersion)*(sectionload.ContractVersion-GeneratorContractVersion)]struct{}{}
