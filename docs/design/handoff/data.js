// Realistic mock MIB data — shaped like A10-AX-MIB sections
// Structure: each node has { name, oid, kind, type?, access?, status?, desc?, indexes?, enum?, children? }
// kind: module | object | scalar | table | entry | column | notification | group

window.MIB_DATA = {
  module: {
    name: "A10-AX-MIB",
    oid: "1.3.6.1.4.1.22610.2.4",
    desc: "Management root OID for the application acceleration family appliance.",
    organization: "A10 Networks",
    contact: "support@a10networks.com",
    lastUpdated: "2024-08-12 18:30Z",
    imported: [
      { name: "A10-COMMON-MIB", items: ["axMgmt"] },
      { name: "IF-MIB", items: ["InterfaceIndex", "ifIndex"] },
      { name: "INET-ADDRESS-MIB", items: ["InetAddressType", "InetAddress"] },
      { name: "SNMPv2-SMI", items: ["MODULE-IDENTITY", "OBJECT-TYPE", "Counter32", "Counter64", "Integer32", "Gauge32", "Unsigned32", "IpAddress", "TimeTicks"] },
      { name: "SNMPv2-TC", items: ["DisplayString", "PhysAddress", "TruthValue", "MacAddress"] },
    ],
  },
  tree: {
    name: "axMgmt", oid: "1.3.6.1.4.1.22610.2.4", kind: "object",
    desc: "Root of A10 Networks application acceleration management subtree.",
    children: [
      {
        name: "axSystem", oid: "1.3.6.1.4.1.22610.2.4.1", kind: "object",
        desc: "Top-level system information group: version, memory, cpu, disk, hardware health.",
        children: [
          {
            name: "axSysVersion", oid: "1.3.6.1.4.1.22610.2.4.1.1", kind: "object",
            desc: "Software image versions on disk and compact flash.",
            children: [
              { name: "axSysPrimaryVersionOnDisk", oid: "1.3.6.1.4.1.22610.2.4.1.1.1", kind: "scalar", type: "DisplayString", access: "read-only", status: "current",
                desc: "Primary software image version stored on the hard disk." },
              { name: "axSysSecondaryVersionOnDisk", oid: "1.3.6.1.4.1.22610.2.4.1.1.2", kind: "scalar", type: "DisplayString", access: "read-only", status: "current",
                desc: "Secondary (fallback) software image version stored on the hard disk." },
              { name: "axSysPrimaryVersionOnCF", oid: "1.3.6.1.4.1.22610.2.4.1.1.3", kind: "scalar", type: "DisplayString", access: "read-only", status: "current",
                desc: "Primary software image version on the compact flash." },
              { name: "axSysSecondaryVersionOnCF", oid: "1.3.6.1.4.1.22610.2.4.1.1.4", kind: "scalar", type: "DisplayString", access: "read-only", status: "current",
                desc: "Secondary software image version on the compact flash." },
            ],
          },
          {
            name: "axSysMemory", oid: "1.3.6.1.4.1.22610.2.4.1.2", kind: "object",
            desc: "Total and used memory counters for the appliance control plane.",
            children: [
              { name: "axSysMemoryTotal", oid: "1.3.6.1.4.1.22610.2.4.1.2.1", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "KB",
                desc: "Total physical memory installed on the system, in kilobytes." },
              { name: "axSysMemoryUsage", oid: "1.3.6.1.4.1.22610.2.4.1.2.2", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "KB",
                desc: "Currently used memory, in kilobytes. Sampled every 5 seconds." },
            ],
          },
          {
            name: "axSysCpu", oid: "1.3.6.1.4.1.22610.2.4.1.3", kind: "object",
            desc: "Per-core and aggregate CPU utilization counters.",
            children: [
              { name: "axSysCpuNumber", oid: "1.3.6.1.4.1.22610.2.4.1.3.1", kind: "scalar", type: "Integer32", access: "read-only", status: "current",
                desc: "Number of CPU cores active on this device." },
              {
                name: "axSysCpuTable", oid: "1.3.6.1.4.1.22610.2.4.1.3.2", kind: "table",
                desc: "One row per CPU core showing instantaneous and averaged utilization.",
                children: [
                  {
                    name: "axSysCpuEntry", oid: "1.3.6.1.4.1.22610.2.4.1.3.2.1", kind: "entry",
                    indexes: ["axSysCpuIndex"],
                    desc: "A row describing a single CPU core.",
                    children: [
                      { name: "axSysCpuIndex", oid: "1.3.6.1.4.1.22610.2.4.1.3.2.1.1", kind: "column", type: "Integer32", access: "not-accessible", status: "current",
                        indexFor: "axSysCpuTable",
                        desc: "Index identifying a single CPU core. Not accessible (used as table index)." },
                      { name: "axSysCpuUsage", oid: "1.3.6.1.4.1.22610.2.4.1.3.2.1.2", kind: "column", type: "Gauge32", access: "read-only", status: "current", units: "percent",
                        desc: "Instantaneous CPU usage for this core, expressed as a percentage (0-100)." },
                      { name: "axSysCpuUsageValue", oid: "1.3.6.1.4.1.22610.2.4.1.3.2.1.3", kind: "column", type: "Integer32", access: "read-only", status: "current",
                        desc: "Raw CPU usage sample for this core." },
                      { name: "axSysCpuCtrlCpuFlag", oid: "1.3.6.1.4.1.22610.2.4.1.3.2.1.4", kind: "column", type: "Integer32", access: "read-only", status: "current",
                        enumVals: [{v:0,n:"dataPlane"},{v:1,n:"controlPlane"}],
                        desc: "Indicates whether this core is a control-plane CPU (1) or data-plane CPU (0)." },
                    ],
                  },
                ],
              },
              { name: "axSysAverageCpuUsage", oid: "1.3.6.1.4.1.22610.2.4.1.3.3", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "percent",
                desc: "Aggregate CPU usage averaged across all cores." },
              { name: "axSysAverageControlCpuUsage", oid: "1.3.6.1.4.1.22610.2.4.1.3.4", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "percent",
                desc: "Average CPU usage across control-plane cores only." },
              { name: "axSysAverageDataCpuUsage", oid: "1.3.6.1.4.1.22610.2.4.1.3.5", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "percent",
                desc: "Average CPU usage across data-plane cores only." },
              {
                name: "axSysCpuUsageTable", oid: "1.3.6.1.4.1.22610.2.4.1.3.6", kind: "table",
                desc: "Per-core CPU usage sampled over multiple windows (1s, 5s, 1m, 5m).",
                children: [
                  {
                    name: "axSysCpuUsageEntry", oid: "1.3.6.1.4.1.22610.2.4.1.3.6.1", kind: "entry",
                    indexes: ["axSysCpuIndexInUsage", "axSysCpuUsagePeriodIndex"],
                    desc: "A single CPU/window sample.",
                    children: [
                      { name: "axSysCpuIndexInUsage", oid: "1.3.6.1.4.1.22610.2.4.1.3.6.1.1", kind: "column", type: "Integer32", access: "not-accessible", status: "current",
                        indexFor: "axSysCpuUsageTable", desc: "CPU core index." },
                      { name: "axSysCpuUsagePeriodIndex", oid: "1.3.6.1.4.1.22610.2.4.1.3.6.1.2", kind: "column", type: "Integer32", access: "not-accessible", status: "current",
                        indexFor: "axSysCpuUsageTable",
                        enumVals: [{v:1,n:"sec1"},{v:2,n:"sec5"},{v:3,n:"min1"},{v:4,n:"min5"}],
                        desc: "Sampling window: 1=1s, 2=5s, 3=1m, 4=5m." },
                      { name: "axSysCpuUsageValueAtPeriod", oid: "1.3.6.1.4.1.22610.2.4.1.3.6.1.3", kind: "column", type: "Gauge32", access: "read-only", status: "current", units: "percent",
                        desc: "CPU usage value sampled over the given window." },
                      { name: "axSysCpuUsageCtrlCpuFlag", oid: "1.3.6.1.4.1.22610.2.4.1.3.6.1.4", kind: "column", type: "Integer32", access: "read-only", status: "current",
                        desc: "Whether the core is control-plane (1) or data-plane (0)." },
                    ],
                  },
                ],
              },
            ],
          },
          {
            name: "axSysDisk", oid: "1.3.6.1.4.1.22610.2.4.1.4", kind: "object",
            desc: "Disk capacity counters.",
            children: [
              { name: "axSysDiskTotalSpace", oid: "1.3.6.1.4.1.22610.2.4.1.4.1", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "MB",
                desc: "Total disk space, in megabytes." },
              { name: "axSysDiskFreeSpace", oid: "1.3.6.1.4.1.22610.2.4.1.4.2", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "MB",
                desc: "Free disk space, in megabytes." },
            ],
          },
          {
            name: "axSysHwInfo", oid: "1.3.6.1.4.1.22610.2.4.1.5", kind: "object",
            desc: "Physical sensors: temperature, voltage, fan speed.",
            children: [
              { name: "axSysHwPhySystemTemp", oid: "1.3.6.1.4.1.22610.2.4.1.5.1", kind: "scalar", type: "Integer32", access: "read-only", status: "current", units: "celsius",
                desc: "System ambient temperature, in degrees Celsius." },
              { name: "axPowerSupplyVoltageTotal", oid: "1.3.6.1.4.1.22610.2.4.1.5.5", kind: "scalar", type: "Integer32", access: "read-only", status: "current",
                desc: "Total active power supply voltage rails count." },
              {
                name: "axPowerSupplyVoltageTable", oid: "1.3.6.1.4.1.22610.2.4.1.5.5.1", kind: "table",
                desc: "Per-rail PSU voltage readings.",
                children: [
                  {
                    name: "axPowerSupplyVoltageEntry", oid: "1.3.6.1.4.1.22610.2.4.1.5.5.1.1", kind: "entry",
                    indexes: ["axPowerSupplyVoltageIndex"],
                    desc: "A single voltage rail row.",
                    children: [
                      { name: "axPowerSupplyVoltageIndex", oid: "1.3.6.1.4.1.22610.2.4.1.5.5.1.1.1", kind: "column", type: "Integer32", access: "not-accessible", status: "current", desc: "Rail index." },
                      { name: "axPowerSupplyVoltageName", oid: "1.3.6.1.4.1.22610.2.4.1.5.5.1.1.2", kind: "column", type: "DisplayString", access: "read-only", status: "current", desc: "Rail label, e.g. \"+12V\", \"+5V\"." },
                      { name: "axPowerSupplyVoltageValue", oid: "1.3.6.1.4.1.22610.2.4.1.5.5.1.1.3", kind: "column", type: "Integer32", access: "read-only", status: "current", units: "millivolts", desc: "Measured voltage in millivolts." },
                    ],
                  },
                ],
              },
            ],
          },
        ],
      },
      {
        name: "axNetwork", oid: "1.3.6.1.4.1.22610.2.4.3", kind: "object",
        desc: "Network-layer statistics: TCP, UDP, NAT, sessions, SSL.",
        children: [
          {
            name: "axNetStatTCPSynRcv", oid: "1.3.6.1.4.1.22610.2.4.3.11.3", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Total TCP SYN packets received since last counter reset.",
          },
          {
            name: "axNetStatNoVportDrop", oid: "1.3.6.1.4.1.22610.2.4.3.11.11", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Packets dropped because no virtual port matched.",
          },
          {
            name: "axNetStatNoSynPktDrop", oid: "1.3.6.1.4.1.22610.2.4.3.11.12", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Packets dropped because they were not SYN on a new flow.",
          },
          {
            name: "axNetStatConnLimitDrop", oid: "1.3.6.1.4.1.22610.2.4.3.11.13", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Connections dropped due to connection-limit policy.",
          },
          {
            name: "axNetStatConnLimitReset", oid: "1.3.6.1.4.1.22610.2.4.3.11.14", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Connections reset due to connection-limit policy.",
          },
          {
            name: "axNetStatProxyNoSockDrop", oid: "1.3.6.1.4.1.22610.2.4.3.11.15", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Proxy drops because no socket was available.",
          },
          {
            name: "axNetStatAflexDrop", oid: "1.3.6.1.4.1.22610.2.4.3.11.16", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Drops triggered by aFleX scripts.",
          },
          {
            name: "axNetStatSessionAgingOut", oid: "1.3.6.1.4.1.22610.2.4.3.11.17", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Sessions that aged out and were removed.",
          },
          {
            name: "axNetStatTCPNoSLB", oid: "1.3.6.1.4.1.22610.2.4.3.11.18", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "TCP packets that did not match any SLB virtual server.",
          },
          {
            name: "axNetStatUDPNoSLB", oid: "1.3.6.1.4.1.22610.2.4.3.11.19", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "UDP packets that did not match any SLB virtual server.",
          },
          {
            name: "axNetStatTCPOutRst", oid: "1.3.6.1.4.1.22610.2.4.3.11.20", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound TCP RST packets sent by the appliance.",
          },
          {
            name: "axNetStatTCPOutRstNoSYN", oid: "1.3.6.1.4.1.22610.2.4.3.11.21", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent because data arrived for a flow without a SYN.",
          },
          {
            name: "axNetStatTCPOutRstL4Proxy", oid: "1.3.6.1.4.1.22610.2.4.3.11.22", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent by the L4 proxy.",
          },
          {
            name: "axNetStatTCPOutRstACKAttack", oid: "1.3.6.1.4.1.22610.2.4.3.11.23", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent in response to ACK-flood mitigation.",
          },
          {
            name: "axNetStatTCPOutRstAFleX", oid: "1.3.6.1.4.1.22610.2.4.3.11.24", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent by aFleX scripts.",
          },
          {
            name: "axNetStatTCPOutRstStaleSess", oid: "1.3.6.1.4.1.22610.2.4.3.11.25", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent for stale sessions.",
          },
          {
            name: "axNetStatTCPOutRstProxy", oid: "1.3.6.1.4.1.22610.2.4.3.11.26", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "Outbound RSTs sent by the proxy stack.",
          },
          {
            name: "axNetStatNoSYNPktDropFIN", oid: "1.3.6.1.4.1.22610.2.4.3.11.27", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "FIN packets dropped because no SYN was seen first." },
          {
            name: "axNetStatNoSYNPktDropRST", oid: "1.3.6.1.4.1.22610.2.4.3.11.28", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "RST packets dropped because no SYN was seen first." },
          {
            name: "axNetStatNoSYNPktDropACK", oid: "1.3.6.1.4.1.22610.2.4.3.11.29", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "ACK packets dropped because no SYN was seen first." },
          {
            name: "axNetStatSYNThrottle", oid: "1.3.6.1.4.1.22610.2.4.3.11.30", kind: "scalar", type: "Counter32", access: "read-only", status: "current",
            desc: "SYN packets that triggered throttling." },
        ],
      },
      {
        name: "axNetStatInsideOutsideTable", oid: "1.3.6.1.4.1.22610.2.4.3.18.2.1", kind: "table",
        desc: "Per-interface inside/outside NAT state counters.",
        children: [
          {
            name: "axNetStatInsideOutsideEntry", oid: "1.3.6.1.4.1.22610.2.4.3.18.2.1.1", kind: "entry",
            indexes: ["axNetStatInsideOutsideIntfIndex"],
            desc: "One row per interface.",
            children: [
              { name: "axNetStatInsideOutsideIntfIndex", oid: "1.3.6.1.4.1.22610.2.4.3.18.2.1.1.1", kind: "column", type: "Integer32", access: "not-accessible", status: "current", desc: "Interface index identifying this row." },
              { name: "axNetStatInsideOutsideIntfName", oid: "1.3.6.1.4.1.22610.2.4.3.18.2.1.1.2", kind: "column", type: "DisplayString", access: "read-only", status: "current", desc: "Friendly interface name, e.g. \"ethernet1\"." },
              { name: "axNetStatInsideOutsideIntfDirection", oid: "1.3.6.1.4.1.22610.2.4.3.18.2.1.1.3", kind: "column", type: "Integer32", access: "read-only", status: "current",
                enumVals: [{v:1,n:"inside"},{v:2,n:"outside"}],
                desc: "Direction relative to NAT: 1=inside, 2=outside." },
            ],
          },
        ],
      },
      {
        name: "axNotifications", oid: "1.3.6.1.4.1.22610.2.4.99", kind: "object",
        desc: "SNMP notifications emitted by the appliance.",
        children: [
          { name: "axCpuUsageHigh", oid: "1.3.6.1.4.1.22610.2.4.99.1", kind: "notification", status: "current",
            objects: ["axSysAverageCpuUsage", "axSysCpuUsage"],
            desc: "Emitted when average CPU usage exceeds the configured threshold for 60 seconds." },
          { name: "axMemoryUsageHigh", oid: "1.3.6.1.4.1.22610.2.4.99.2", kind: "notification", status: "current",
            objects: ["axSysMemoryTotal", "axSysMemoryUsage"],
            desc: "Emitted when memory usage exceeds the configured threshold." },
          { name: "axDiskFreeLow", oid: "1.3.6.1.4.1.22610.2.4.99.3", kind: "notification", status: "current",
            objects: ["axSysDiskTotalSpace", "axSysDiskFreeSpace"],
            desc: "Emitted when free disk space drops below the configured floor." },
          { name: "axTempHigh", oid: "1.3.6.1.4.1.22610.2.4.99.4", kind: "notification", status: "current",
            objects: ["axSysHwPhySystemTemp"],
            desc: "Emitted when system temperature exceeds the configured ceiling." },
        ],
      },
    ],
  },
};
