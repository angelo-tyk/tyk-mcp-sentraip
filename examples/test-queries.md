# Test Queries for Claude MCP Integration

## Basic Threat Queries

1. **Single IP Check**
"Check the reputation of IP address 185.220.101.45"

2. **Multiple IP Analysis**  
"Analyze these IPs for threats: 185.220.101.45, 192.168.1.100, 8.8.8.8"

## Advanced Security Queries

3. **Real-time Threat Detection**
"Who is attacking our gateway right now?"

4. **Pattern Analysis**
"Show me VPN traffic trends from suspicious sources in the last 2 hours"

5. **Geographic Analysis**
"What's the threat profile for Russian traffic today?"

6. **Automated Response**
"Find all malicious IPs targeting our /api/payment endpoint and suggest blocking actions"

## Investigation Queries

7. **Incident Analysis**
"Investigate recent attacks from the 185.220.101.0/24 network range"

8. **Trend Analysis**
"Compare today's bot traffic to last week's patterns"

9. **False Positive Check**
"Are there any legitimate users being blocked from VPN IPs?"

10. **Compliance Reporting**
 ```
 "Generate a security summary for the last 24 hours including blocked threats and false positive rates"
 ```

## Expected Responses

The Claude integration should provide:
- Real threat intelligence data from SentraIP
- Natural language explanations of security events
- Actionable recommendations for threat mitigation
- Context-aware analysis combining multiple data sources
- Audit-friendly detailed logs and timestamps
