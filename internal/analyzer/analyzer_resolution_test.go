package analyzer

import (
	"testing"
)

func TestAnalyzeUsesPackageQualifiedStructFields(t *testing.T) {
	flows, err := Analyze([]string{"testdata/package_scopes"}, Options{
		RPC:   "RunAlpha",
		Depth: 3,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	repoCall := findInterfaceCall(flow.Trail.InterfaceCalls, "uc.repo.Store")
	if repoCall == nil {
		t.Fatalf("interface calls = %#v, want uc.repo.Store", flow.Trail.InterfaceCalls)
	}
	if repoCall.Interface != "AlphaStore" {
		t.Fatalf("repository interface = %q, want AlphaStore", repoCall.Interface)
	}
	if len(repoCall.Implementations) != 1 {
		t.Fatalf("repository implementations = %#v, want one candidate", repoCall.Implementations)
	}
	if got := repoCall.Implementations[0].Call.Symbol; got != "AlphaStore.Store" {
		t.Fatalf("repository implementation = %q, want AlphaStore.Store", got)
	}
	if hasCallImplementation(repoCall.Implementations, "BetaStore.Store") {
		t.Fatalf("repository implementations = %#v, must not include BetaStore.Store", repoCall.Implementations)
	}
}

func TestAnalyzeFollowsConstructorChainedAndLocalVariableCalls(t *testing.T) {
	flows, err := Analyze([]string{"testdata/chained_resolution"}, Options{Depth: 4})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	usecases := flow.Trail.LayerCalls("usecase")
	if !hasCall(usecases, "fooUsecase.GetFoo") {
		t.Fatalf("usecases = %#v, want fooUsecase.GetFoo", usecases)
	}
	services := flow.Trail.LayerCalls("service")
	if !hasCall(services, "fooService.FetchFoo") {
		t.Fatalf("services = %#v, want fooService.FetchFoo", services)
	}
	if !hasCall(usecases, "NewFooUsecase().GetFoo") {
		t.Fatalf("usecases = %#v, want NewFooUsecase().GetFoo", usecases)
	}
}

func TestAnalyzeNarrowsInterfaceCandidatesFromConstructorDI(t *testing.T) {
	flows, err := Analyze([]string{"testdata/constructor_di"}, Options{Depth: 3})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	workflowCall := findInterfaceCall(flow.Trail.InterfaceCalls, "s.workflow.Run")
	if workflowCall == nil {
		t.Fatalf("interface calls = %#v, want s.workflow.Run", flow.Trail.InterfaceCalls)
	}
	if len(workflowCall.Implementations) != 1 {
		t.Fatalf("workflow implementations = %#v, want one constructor-selected candidate", workflowCall.Implementations)
	}
	if got := workflowCall.Implementations[0].Call.Symbol; got != "createUsecase.Run" {
		t.Fatalf("workflow implementation = %q, want createUsecase.Run", got)
	}
	if hasCallImplementation(workflowCall.Implementations, "archiveUsecase.Run") {
		t.Fatalf("workflow implementations = %#v, must not include archiveUsecase.Run", workflowCall.Implementations)
	}

	repoCall := findInterfaceCall(flow.Trail.InterfaceCalls, "c.repo.Save")
	if repoCall == nil {
		t.Fatalf("interface calls = %#v, want c.repo.Save", flow.Trail.InterfaceCalls)
	}
	if len(repoCall.Implementations) != 1 {
		t.Fatalf("repository implementations = %#v, want one constructor-selected candidate", repoCall.Implementations)
	}
	if got := repoCall.Implementations[0].Call.Symbol; got != "sqlRepository.Save" {
		t.Fatalf("repository implementation = %q, want sqlRepository.Save", got)
	}
	if hasCallImplementation(repoCall.Implementations, "memoryRepository.Save") {
		t.Fatalf("repository implementations = %#v, must not include memoryRepository.Save", repoCall.Implementations)
	}
}

func TestAnalyzeFallsBackToReceiverNameWhenInterfaceMethodSetIsIncomplete(t *testing.T) {
	flows, err := Analyze([]string{"testdata/interface_fallback"}, Options{Depth: 3})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	gatewayCall := findInterfaceCall(flow.Trail.InterfaceCalls, "w.gateway.Run")
	if gatewayCall == nil {
		t.Fatalf("interface calls = %#v, want w.gateway.Run", flow.Trail.InterfaceCalls)
	}
	if len(gatewayCall.Implementations) != 1 {
		t.Fatalf("gateway implementations = %#v, want fallback candidate", gatewayCall.Implementations)
	}
	if got := gatewayCall.Implementations[0].Call.Symbol; got != "remoteGatewayClient.Run" {
		t.Fatalf("gateway implementation = %q, want remoteGatewayClient.Run", got)
	}
}

func TestAnalyzeDoesNotTreatQualifiedConcreteTypeAsSameNamedInterface(t *testing.T) {
	flows, err := Analyze([]string{"testdata/qualified_types"}, Options{Depth: 2})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if call := findInterfaceCall(flow.Trail.InterfaceCalls, "payload.Token"); call != nil {
		t.Fatalf("payload.Token was resolved as interface call: %#v", call)
	}
	eventCall := findInterfaceCall(flow.Trail.InterfaceCalls, "w.events.Send")
	if eventCall == nil {
		t.Fatalf("interface calls = %#v, want w.events.Send", flow.Trail.InterfaceCalls)
	}
	if len(eventCall.Implementations) != 1 {
		t.Fatalf("event implementations = %#v, want one asserted cross-package candidate", eventCall.Implementations)
	}
	if got := eventCall.Implementations[0].Call.Symbol; got != "Sender.Send" {
		t.Fatalf("event implementation = %q, want Sender.Send", got)
	}
}

func TestAnalyzePropagatesConcreteTypesThroughConstructorParameters(t *testing.T) {
	flows, err := Analyze([]string{"testdata/constructor_param_flow"}, Options{Depth: 2})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	workflowCall := findInterfaceCall(flow.Trail.InterfaceCalls, "s.Workflow.Run")
	if workflowCall == nil {
		t.Fatalf("interface calls = %#v, want s.Workflow.Run", flow.Trail.InterfaceCalls)
	}
	if len(workflowCall.Implementations) != 1 {
		t.Fatalf("workflow implementations = %#v, want constructor-propagated candidate", workflowCall.Implementations)
	}
	if got := workflowCall.Implementations[0].Call.Symbol; got != "workflowImpl.Run" {
		t.Fatalf("workflow implementation = %q, want workflowImpl.Run", got)
	}
}

func TestAnalyzeFollowsFunctionValuesPassedAsArguments(t *testing.T) {
	flows, err := Analyze([]string{"testdata/function_values"}, Options{Depth: 3})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	trace := findFunctionValue(flow.Trail.FunctionValues, "InvokeWithToken", "s.processor.Run")
	if trace == nil {
		t.Fatalf("function values = %#v, want InvokeWithToken -> s.processor.Run", flow.Trail.FunctionValues)
	}
	if len(trace.Implementations) != 1 {
		t.Fatalf("function value implementations = %#v, want one implementation", trace.Implementations)
	}
	if got := trace.Implementations[0].Call.Symbol; got != "sampleUsecase.Run" {
		t.Fatalf("function value implementation = %q, want sampleUsecase.Run", got)
	}
	standaloneTrace := findFunctionValue(flow.Trail.FunctionValues, "InvokeWithToken", "runStandalone")
	if standaloneTrace == nil {
		t.Fatalf("function values = %#v, want InvokeWithToken -> runStandalone", flow.Trail.FunctionValues)
	}
	if len(standaloneTrace.Implementations) != 1 {
		t.Fatalf("standalone function implementations = %#v, want one implementation", standaloneTrace.Implementations)
	}
	if got := standaloneTrace.Implementations[0].Call.Symbol; got != "runStandalone" {
		t.Fatalf("standalone function implementation = %q, want runStandalone", got)
	}

	usecases := flow.Trail.LayerCalls("usecase")
	if hasCall(usecases, "s.processor.Run") {
		t.Fatalf("usecases = %#v, must not include function value callsite s.processor.Run", usecases)
	}
	if !hasCall(usecases, "sampleUsecase.Run") {
		t.Fatalf("usecases = %#v, want implementation sampleUsecase.Run", usecases)
	}
	repositories := flow.Trail.LayerCalls("repository")
	if !hasCall(repositories, "u.repo.Save") {
		t.Fatalf("repositories = %#v, want u.repo.Save from function value implementation", repositories)
	}
	if !hasCall(repositories, "sampleRepository.Save") {
		t.Fatalf("repositories = %#v, want sampleRepository.Save from function value implementation", repositories)
	}
}
