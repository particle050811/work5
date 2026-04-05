package response

import "testing"

func TestSuccess(t *testing.T) {
	resp := Success()
	if resp.Code != CodeSuccess || resp.Msg != "成功" {
		t.Fatalf("Success() = %+v", resp)
	}

	resp = Success("注册成功")
	if resp.Code != CodeSuccess || resp.Msg != "注册成功" {
		t.Fatalf("Success(custom) = %+v", resp)
	}
}

func TestError(t *testing.T) {
	resp := Error(CodeUserNotFound)
	if resp.Code != CodeUserNotFound || resp.Msg != "用户不存在" {
		t.Fatalf("Error() = %+v", resp)
	}

	resp = Error(CodeUserNotFound, "自定义消息")
	if resp.Code != CodeUserNotFound || resp.Msg != "自定义消息" {
		t.Fatalf("Error(custom) = %+v", resp)
	}

	resp = Error(9999)
	if resp.Code != 9999 || resp.Msg != "未知错误" {
		t.Fatalf("Error(unknown) = %+v", resp)
	}
}

func TestHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  int32
		want int32
	}{
		{name: "ParamError", got: ParamError().Code, want: CodeParamError},
		{name: "Unauthorized", got: Unauthorized().Code, want: CodeUnauthorized},
		{name: "Forbidden", got: Forbidden().Code, want: CodeForbidden},
		{name: "NotFound", got: NotFound().Code, want: CodeNotFound},
		{name: "InternalError", got: InternalError().Code, want: CodeInternalError},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Fatalf("%s code = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
