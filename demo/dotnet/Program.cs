using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Security.Claims;
using System.Text.Json.Serialization;
using Microsoft.AspNetCore.Authentication.JwtBearer;

var settings = DemoSettings.Load();
var builder = WebApplication.CreateBuilder(args);

builder.WebHost.UseUrls(settings.Address);
builder.Services.AddSingleton(settings);
builder.Services
    .AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.Authority = settings.OidcIssuer;
        options.Audience = settings.OidcAudience;
        options.MapInboundClaims = false;
        options.RequireHttpsMetadata = settings.RequireHttpsMetadata;
    });
builder.Services.AddAuthorization();
builder.Services.AddHttpClient<PdpClient>(client => client.Timeout = settings.PdpTimeout);
builder.Services.AddScoped<PdpAuthorizationFilter>();

var app = builder.Build();
app.UseAuthentication();
app.UseAuthorization();

app.MapGet("/healthz", () => Results.Ok(new { status = "ok" })).AllowAnonymous();
app.MapGet("/openapi.json", () => Results.Text(OpenApiDocument.Value, "application/json")).AllowAnonymous();
app.MapGet("/api/orders/{id}", (string id) => Results.Ok(new
{
    id,
    number = "ORD-2026-0001",
    status = "paid"
}))
    .RequireAuthorization()
    .AddEndpointFilter<PdpAuthorizationFilter>();

app.Run();

sealed class PdpAuthorizationFilter(
    PdpClient pdp,
    DemoSettings settings,
    ILogger<PdpAuthorizationFilter> logger) : IEndpointFilter
{
    public async ValueTask<object?> InvokeAsync(EndpointFilterInvocationContext context, EndpointFilterDelegate next)
    {
        // 用户标识和租户只能从已验证的 JWT claims 读取，不能信任请求体传入的值。
        var httpContext = context.HttpContext;
        var subjectId = httpContext.User.FindFirstValue("sub");
        var tenantId = httpContext.User.FindFirstValue(settings.TenantClaim);
        if (string.IsNullOrWhiteSpace(subjectId) || string.IsNullOrWhiteSpace(tenantId))
        {
            return Results.Unauthorized();
        }

        var attributes = settings.SubjectAttributeClaims
            .Select(name => (Name: name, Value: httpContext.User.FindFirstValue(name)))
            .Where(item => item.Value is not null)
            .ToDictionary(item => item.Name, item => (object)item.Value!, StringComparer.Ordinal);
        var requestId = httpContext.Request.Headers["X-Request-ID"].FirstOrDefault();
        if (string.IsNullOrWhiteSpace(requestId))
        {
            requestId = Guid.NewGuid().ToString("N");
        }

        PdpDecisionResponse decision;
        try
        {
            decision = await pdp.DecideAsync(
                subjectId,
                tenantId,
                attributes,
                httpContext.Request.Method,
                requestId,
                httpContext.RequestAborted);
        }
        catch (Exception exception)
        {
            logger.LogError(exception, "PDP request {RequestId} failed", requestId);
            return Results.StatusCode(StatusCodes.Status503ServiceUnavailable);
        }

        if (!string.Equals(decision.Decision, "allow", StringComparison.Ordinal))
        {
            return Results.Forbid();
        }

        httpContext.Response.Headers["X-Request-ID"] = requestId;
        return await next(context);
    }
}

sealed class PdpClient(HttpClient httpClient, DemoSettings settings)
{
    public async Task<PdpDecisionResponse> DecideAsync(
        string subjectId,
        string tenantId,
        IReadOnlyDictionary<string, object> attributes,
        string method,
        string requestId,
        CancellationToken cancellationToken)
    {
        // 当前路由是 API 级鉴权，因此固定发送 api；业务对象级路由应由服务端固定发送 data。
        var input = new PdpDecisionRequest(
            requestId,
            "api",
            settings.ServiceResource,
            tenantId,
            new PdpSubject(subjectId, attributes.Count == 0 ? null : attributes),
            new PdpAction("http", method),
            new PdpResource("api_endpoint", settings.EndpointId));
        using var request = new HttpRequestMessage(HttpMethod.Post, settings.PdpDecisionUrl)
        {
            Content = JsonContent.Create(input)
        };
        request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", settings.PdpServiceCredential);

        // 直接调用权限中心 HTTP API，不依赖 SDK。网络或协议异常都会阻止业务接口继续执行。
        using var response = await httpClient.SendAsync(request, cancellationToken);
        if (!response.IsSuccessStatusCode)
        {
            throw new HttpRequestException($"Permission center returned HTTP {(int)response.StatusCode}.");
        }

        var decision = await response.Content.ReadFromJsonAsync<PdpDecisionResponse>(cancellationToken);
        if (decision is null || (decision.Decision != "allow" && decision.Decision != "deny"))
        {
            throw new InvalidOperationException("Permission center returned an invalid decision.");
        }

        return decision;
    }
}

sealed record PdpDecisionRequest(
    [property: JsonPropertyName("request_id")] string RequestId,
    [property: JsonPropertyName("authorization_type")] string AuthorizationType,
    [property: JsonPropertyName("service_resource")] string ServiceResource,
    [property: JsonPropertyName("tenant_id")] string TenantId,
    [property: JsonPropertyName("subject")] PdpSubject Subject,
    [property: JsonPropertyName("action")] PdpAction Action,
    [property: JsonPropertyName("resource")] PdpResource Resource);

sealed record PdpSubject(
    [property: JsonPropertyName("id")] string Id,
    [property: JsonPropertyName("attributes"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] IReadOnlyDictionary<string, object>? Attributes);

sealed record PdpAction(
    [property: JsonPropertyName("kind")] string Kind,
    [property: JsonPropertyName("method")] string Method);

sealed record PdpResource(
    [property: JsonPropertyName("kind")] string Kind,
    [property: JsonPropertyName("endpoint_id")] Guid EndpointId);

sealed record PdpDecisionResponse(
    [property: JsonPropertyName("decision")] string Decision,
    [property: JsonPropertyName("reason_code")] string? ReasonCode);

sealed record DemoSettings(
    string Address,
    Uri PdpDecisionUrl,
    string PdpServiceCredential,
    string ServiceResource,
    Guid EndpointId,
    string OidcIssuer,
    string OidcAudience,
    string TenantClaim,
    string[] SubjectAttributeClaims,
    TimeSpan PdpTimeout,
    bool RequireHttpsMetadata)
{
    public static DemoSettings Load()
    {
        var permissionCenterUrl = Required("DEMO_PERMISSION_CENTER_URL").TrimEnd('/');
        if (!Uri.TryCreate($"{permissionCenterUrl}/v1/pdp/decisions", UriKind.Absolute, out var decisionUrl)
            || (decisionUrl.Scheme != Uri.UriSchemeHttp && decisionUrl.Scheme != Uri.UriSchemeHttps))
        {
            throw new InvalidOperationException("DEMO_PERMISSION_CENTER_URL must be an absolute HTTP(S) URL.");
        }

        if (!Guid.TryParse(Required("DEMO_ENDPOINT_ID"), out var endpointId) || endpointId == Guid.Empty)
        {
            throw new InvalidOperationException("DEMO_ENDPOINT_ID must be the endpoint UUID returned by permission center.");
        }

        if (!TimeSpan.TryParse(Environment.GetEnvironmentVariable("DEMO_PDP_TIMEOUT") ?? "00:00:02", out var timeout)
            || timeout <= TimeSpan.Zero)
        {
            throw new InvalidOperationException("DEMO_PDP_TIMEOUT must be a positive TimeSpan.");
        }

        return new DemoSettings(
            Environment.GetEnvironmentVariable("DEMO_ADDR") ?? "http://localhost:8091",
            decisionUrl,
            Required("DEMO_PDP_SERVICE_CREDENTIAL"),
            Required("DEMO_SERVICE_RESOURCE"),
            endpointId,
            Required("DEMO_OIDC_ISSUER"),
            Required("DEMO_OIDC_AUDIENCE"),
            Environment.GetEnvironmentVariable("DEMO_TENANT_CLAIM") ?? "tenant_id",
            (Environment.GetEnvironmentVariable("DEMO_SUBJECT_ATTRIBUTE_CLAIMS") ?? "department,account_status")
                .Split(',', StringSplitOptions.TrimEntries | StringSplitOptions.RemoveEmptyEntries),
            timeout,
            !string.Equals(Environment.GetEnvironmentVariable("DEMO_OIDC_ALLOW_HTTP"), "true", StringComparison.OrdinalIgnoreCase));
    }

    private static string Required(string name)
    {
        var value = Environment.GetEnvironmentVariable(name)?.Trim();
        return string.IsNullOrEmpty(value)
            ? throw new InvalidOperationException($"{name} is required.")
            : value;
    }
}

static class OpenApiDocument
{
    public const string Value = """
        {
          "openapi": "3.0.3",
          "info": {"title": "Permission Center .NET HTTP Demo", "version": "1.0.0"},
          "paths": {
            "/api/orders/{id}": {
              "get": {
                "tags": ["Orders"],
                "summary": "Get an order",
                "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}],
                "responses": {"200": {"description": "Order"}, "401": {"description": "Unauthenticated"}, "403": {"description": "Forbidden"}}
              }
            }
          }
        }
        """;
}

public partial class Program;
