package grpcapi

import (
	ctx "context"
	pb "github.com/OliveTin/OliveTin/gen/grpc"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"errors"
	"net"
	"sort"

	acl "github.com/OliveTin/OliveTin/internal/acl"
	config "github.com/OliveTin/OliveTin/internal/config"
	executor "github.com/OliveTin/OliveTin/internal/executor"
	installationinfo "github.com/OliveTin/OliveTin/internal/installationinfo"
)

var (
	cfg *config.Config
)

type oliveTinAPI struct {
	// Uncomment this if you want to allow undefined methods during dev.
	//	pb.UnimplementedOliveTinApiServiceServer

	executor *executor.Executor
}

func (api *oliveTinAPI) StartAction(ctx ctx.Context, req *pb.StartActionRequest) (*pb.StartActionResponse, error) {
	args := make(map[string]string)

	for _, arg := range req.Arguments {
		args[arg.Name] = arg.Value
	}

	execReq := executor.ExecutionRequest{
		ActionName:        req.ActionName,
		UUID:              req.Uuid,
		Arguments:         args,
		AuthenticatedUser: acl.UserFromContext(ctx, cfg),
		Cfg:               cfg,
	}

	_, uuid := api.executor.ExecRequest(&execReq)

	return &pb.StartActionResponse{
		ExecutionUuid: uuid,
	}, nil
}

func (api *oliveTinAPI) StartActionAndWait(ctx ctx.Context, req *pb.StartActionAndWaitRequest) (*pb.StartActionAndWaitResponse, error) {
	args := make(map[string]string)

	execReq := executor.ExecutionRequest{
		ActionName:        req.ActionName,
		UUID:              uuid.NewString(),
		Arguments:         args,
		AuthenticatedUser: acl.UserFromContext(ctx, cfg),
		Cfg:               cfg,
	}

	wg, _ := api.executor.ExecRequest(&execReq)
	wg.Wait()

	internalLogEntry, ok := api.executor.Logs[execReq.UUID]

	if ok {
		return &pb.StartActionAndWaitResponse{
			LogEntry: internalLogEntryToPb(internalLogEntry),
		}, nil
	} else {
		return nil, errors.New("Execution not found!")
	}
}

func (api *oliveTinAPI) StartActionByAlias(ctx ctx.Context, req *pb.StartActionByAliasRequest) (*pb.StartActionByAliasResponse, error) {
	args := make(map[string]string)

	action := findActionByAlias(req.ActionAlias)

	if action == nil {
		log.Warnf("ByAlias action alias not found: %v, cannot start execution.", req.ActionAlias)
		return &pb.StartActionByAliasResponse{
			ExecutionUuid: "",
		}, errors.New("ByAlias action alias not found")
	}

	execReq := executor.ExecutionRequest{
		ActionName: action.Title,
		Action:     action,
		UUID:       uuid.NewString(),
		Arguments:  args,
		AuthenticatedUser: &acl.AuthenticatedUser{
			Username:  "webhook",
			Usergroup: "webhook",
		},
		Cfg: cfg,
	}

	_, uuid := api.executor.ExecRequest(&execReq)

	return &pb.StartActionByAliasResponse{
		ExecutionUuid: uuid,
	}, nil
}

func (api *oliveTinAPI) StartActionByAliasAndWait(ctx ctx.Context, req *pb.StartActionByAliasAndWaitRequest) (*pb.StartActionByAliasAndWaitResponse, error) {
	args := make(map[string]string)

	action := findActionByAlias(req.ActionAlias)

	if action == nil {
		log.Warnf("ByAlias action alias not found: %v, cannot start execution.", req.ActionAlias)

		return &pb.StartActionByAliasAndWaitResponse{}, errors.New("ByAlias action alias not found")
	}

	execReq := executor.ExecutionRequest{
		ActionName: action.Title,
		Action:     action,
		UUID:       uuid.NewString(),
		Arguments:  args,
		AuthenticatedUser: &acl.AuthenticatedUser{
			Username:  "webhook",
			Usergroup: "webhook",
		},
		Cfg: cfg,
	}

	wg, _ := api.executor.ExecRequest(&execReq)
	wg.Wait()

	internalLogEntry, ok := api.executor.Logs[execReq.UUID]

	if ok {
		return &pb.StartActionByAliasAndWaitResponse{
			LogEntry: internalLogEntryToPb(internalLogEntry),
		}, nil
	} else {
		return nil, errors.New("Execution not found!")
	}
}

func internalLogEntryToPb(logEntry *executor.InternalLogEntry) *pb.LogEntry {
	return &pb.LogEntry{
		ActionTitle:       logEntry.ActionTitle,
		ActionIcon:        logEntry.ActionIcon,
		DatetimeStarted:   logEntry.DatetimeStarted,
		DatetimeFinished:  logEntry.DatetimeFinished,
		Stdout:            logEntry.Stdout,
		Stderr:            logEntry.Stderr,
		TimedOut:          logEntry.TimedOut,
		Blocked:           logEntry.Blocked,
		ExitCode:          logEntry.ExitCode,
		Tags:              logEntry.Tags,
		ExecutionUuid:     logEntry.UUID,
		ExecutionStarted:  logEntry.ExecutionStarted,
		ExecutionFinished: logEntry.ExecutionFinished,
	}
}

func (api *oliveTinAPI) ExecutionStatus(ctx ctx.Context, req *pb.ExecutionStatusRequest) (*pb.ExecutionStatusResponse, error) {
	res := &pb.ExecutionStatusResponse{}

	logEntry, ok := api.executor.Logs[req.ExecutionUuid]

	if !ok {
		return res, nil
	}

	res.LogEntry = internalLogEntryToPb(logEntry)

	return res, nil
}

/**
func (api *oliveTinAPI) WatchExecution(req *pb.WatchExecutionRequest, srv pb.OliveTinApi_WatchExecutionServer) error {
	log.Infof("Watch")

	if logEntry, ok := api.executor.Logs[req.ExecutionUuid]; !ok {
		log.Errorf("Execution not found: %v", req.ExecutionUuid)

		return nil
	} else {
		if logEntry.ExecutionStarted {
			for !logEntry.ExecutionCompleted {
				tmp := make([]byte, 256)

				red, err := io.ReadAtLeast(logEntry.StdoutBuffer, tmp, 1)

				log.Infof("%v %v", red, err)

				srv.Send(&pb.WatchExecutionUpdate{
					Update: string(tmp),
				})
			}
		}

		return nil
	}
}
*/

func (api *oliveTinAPI) GetDashboardComponents(ctx ctx.Context, req *pb.GetDashboardComponentsRequest) (*pb.GetDashboardComponentsResponse, error) {
	user := acl.UserFromContext(ctx, cfg)

	res := actionsCfgToPb(cfg.Actions, user)

	if len(res.Actions) == 0 {
		log.Warn("Zero actions found - check that you have some actions defined, with a view permission")
	}

	log.Tracef("GetDashboardComponents: %v", res)

	return res, nil
}

func (api *oliveTinAPI) GetLogs(ctx ctx.Context, req *pb.GetLogsRequest) (*pb.GetLogsResponse, error) {
	ret := &pb.GetLogsResponse{}

	// TODO Limit to 10 entries or something to prevent browser lag.

	for uuid, logEntry := range api.executor.Logs {
		pbLogEntry := internalLogEntryToPb(logEntry)
		pbLogEntry.ExecutionUuid = uuid

		ret.Logs = append(ret.Logs, pbLogEntry)
	}

	sorter := func(i, j int) bool {
		return ret.Logs[i].DatetimeStarted < ret.Logs[j].DatetimeStarted
	}

	sort.Slice(ret.Logs, sorter)

	return ret, nil
}

/*
This function is ONLY a helper for the UI - the arguments are validated properly
on the StartAction -> Executor chain. This is here basically to provide helpful
error messages more quickly before starting the action.
*/
func (api *oliveTinAPI) ValidateArgumentType(ctx ctx.Context, req *pb.ValidateArgumentTypeRequest) (*pb.ValidateArgumentTypeResponse, error) {
	err := executor.TypeSafetyCheck("", req.Value, req.Type)
	desc := ""

	if err != nil {
		desc = err.Error()
	}

	return &pb.ValidateArgumentTypeResponse{
		Valid:       err == nil,
		Description: desc,
	}, nil
}

func (api *oliveTinAPI) WhoAmI(ctx ctx.Context, req *pb.WhoAmIRequest) (*pb.WhoAmIResponse, error) {
	user := acl.UserFromContext(ctx, cfg)

	res := &pb.WhoAmIResponse{
		AuthenticatedUser: user.Username,
	}

	log.Warnf("usergroup: %v", user.Usergroup)

	return res, nil
}

func (api *oliveTinAPI) SosReport(ctx ctx.Context, req *pb.SosReportRequest) (*pb.SosReportResponse, error) {
	res := &pb.SosReportResponse{
		Alert: "Your SOS Report has been logged to OliveTin logs.",
	}

	log.Infof("\n" + installationinfo.GetSosReport())

	return res, nil
}

// Start will start the GRPC API.
func Start(globalConfig *config.Config, ex *executor.Executor) {
	cfg = globalConfig

	lis, err := net.Listen("tcp", cfg.ListenAddressGrpcActions)

	if err != nil {
		log.Fatalf("Failed to listen - %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOliveTinApiServiceServer(grpcServer, newServer(ex))

	err = grpcServer.Serve(lis)

	if err != nil {
		log.Fatalf("Could not start gRPC Server - %v", err)
	}
}

func newServer(ex *executor.Executor) *oliveTinAPI {
	server := oliveTinAPI{}
	server.executor = ex
	return &server
}
