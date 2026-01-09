// K6 Workflow Tracker
// Tracks Temporal workflow execution across the order fulfillment lifecycle

import {
  TRACKING_CONFIG,
  getWorkflowStatus,
  getWorkflowDescription,
  getWorkflowSignals,
  queryWorkflow,
} from './tracking.js';

/**
 * WorkflowTracker - Tracks Temporal workflow execution for an order
 * Monitors workflow states, signal delivery, and generates execution reports
 */
export class WorkflowTracker {
  constructor(orderId) {
    this.orderId = orderId;
    this.enabled = TRACKING_CONFIG.enabled;
    this.startTime = Date.now();

    // Map of workflow types to their IDs
    this.workflowIds = {
      orderFulfillment: `order-fulfillment-${orderId}`,
      planning: `planning-${orderId}`,
      picking: `picking-${orderId}`,
      consolidation: `consolidation-${orderId}`,
      packing: `packing-${orderId}`,
      wes: `wes-${orderId}`,
      receiving: `receiving-${orderId}`,
      stow: `stow-${orderId}`,
    };

    // Signal tracking log
    this.signalLog = [];

    // Workflow status history
    this.statusHistory = [];

    // Workflow execution timeline
    this.timeline = [];
  }

  /**
   * Gets the workflow ID for a given workflow type
   * @param {string} workflowType - Type of workflow
   * @returns {string} Workflow ID
   */
  getWorkflowId(workflowType) {
    return this.workflowIds[workflowType] || `${workflowType}-${this.orderId}`;
  }

  /**
   * Gets the status of a specific workflow
   * @param {string} workflowType - Type of workflow to check
   * @returns {Object} Workflow status
   */
  getStatus(workflowType) {
    if (!this.enabled) {
      return { workflowType, skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    const status = getWorkflowStatus(workflowId);

    return {
      workflowType,
      workflowId,
      ...status,
      checkedAt: new Date().toISOString(),
    };
  }

  /**
   * Gets detailed description of a workflow
   * @param {string} workflowType - Type of workflow
   * @returns {Object} Workflow description
   */
  getDescription(workflowType) {
    if (!this.enabled) {
      return { workflowType, skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    return {
      workflowType,
      workflowId,
      ...getWorkflowDescription(workflowId),
    };
  }

  /**
   * Gets statuses of all workflows for this order
   * @returns {Array} Array of workflow statuses
   */
  getAllWorkflowStatuses() {
    if (!this.enabled) {
      return { orderId: this.orderId, skipped: true };
    }

    const statuses = Object.entries(this.workflowIds).map(([type, id]) => {
      const status = getWorkflowStatus(id);
      return {
        workflowType: type,
        workflowId: id,
        ...status,
      };
    });

    const timestamp = new Date().toISOString();

    // Record in history
    this.statusHistory.push({
      timestamp,
      statuses,
    });

    return {
      orderId: this.orderId,
      checkedAt: timestamp,
      workflows: statuses,
      summary: {
        running: statuses.filter(s => s.isRunning).length,
        completed: statuses.filter(s => s.status === 'COMPLETED').length,
        failed: statuses.filter(s => s.status === 'FAILED').length,
        notFound: statuses.filter(s => !s.found).length,
      },
    };
  }

  /**
   * Tracks a signal being sent to a workflow
   * @param {string} signalName - Name of the signal
   * @param {string} workflowType - Type of workflow receiving the signal
   * @param {Object} signalData - Data sent with the signal
   * @returns {Object} Signal tracking result
   */
  trackSignal(signalName, workflowType, signalData = null) {
    if (!this.enabled) {
      return { skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    const timestamp = new Date().toISOString();

    // Get signals before (to verify delivery later)
    const signalsBefore = getWorkflowSignals(workflowId);

    const signalEntry = {
      signalName,
      workflowType,
      workflowId,
      signalData,
      sentAt: timestamp,
      signalCountBefore: signalsBefore.signalCount || 0,
    };

    this.signalLog.push(signalEntry);

    this.timeline.push({
      type: 'signal_sent',
      signal: signalName,
      workflow: workflowType,
      timestamp,
    });

    if (TRACKING_CONFIG.logLevel === 'verbose') {
      console.log(`[WORKFLOW-TRACKER] Signal sent: ${signalName} → ${workflowType} (${workflowId})`);
    }

    return signalEntry;
  }

  /**
   * Verifies that a signal was delivered to a workflow
   * @param {string} signalName - Name of the signal
   * @param {string} workflowType - Type of workflow
   * @returns {Object} Verification result
   */
  verifySignalDelivery(signalName, workflowType) {
    if (!this.enabled) {
      return { skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    const signals = getWorkflowSignals(workflowId);

    const delivered = signals.signals?.some(s => s.name === signalName) || false;
    const timestamp = new Date().toISOString();

    // Update signal log entry
    const logEntry = this.signalLog.find(
      s => s.signalName === signalName && s.workflowType === workflowType && !s.verified
    );

    if (logEntry) {
      logEntry.verified = true;
      logEntry.delivered = delivered;
      logEntry.verifiedAt = timestamp;
    }

    if (TRACKING_CONFIG.logLevel === 'verbose') {
      console.log(`[WORKFLOW-TRACKER] Signal delivery: ${signalName} → ${workflowType} = ${delivered ? 'DELIVERED' : 'NOT FOUND'}`);
    }

    return {
      signalName,
      workflowType,
      workflowId,
      delivered,
      signalCount: signals.signalCount,
      verifiedAt: timestamp,
    };
  }

  /**
   * Waits for a workflow to reach a specific state
   * @param {string} workflowType - Type of workflow
   * @param {string} expectedStatus - Expected status to wait for
   * @param {number} timeoutMs - Timeout in milliseconds
   * @param {number} pollIntervalMs - Polling interval
   * @returns {Object} Wait result
   */
  waitForWorkflowState(workflowType, expectedStatus, timeoutMs = 30000, pollIntervalMs = 1000) {
    if (!this.enabled) {
      return { skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    const startTime = Date.now();
    let attempts = 0;
    let lastStatus = null;

    while (Date.now() - startTime < timeoutMs) {
      attempts++;
      const status = getWorkflowStatus(workflowId);
      lastStatus = status.status;

      if (status.status === expectedStatus) {
        const duration = Date.now() - startTime;

        this.timeline.push({
          type: 'state_reached',
          workflow: workflowType,
          status: expectedStatus,
          duration,
          attempts,
          timestamp: new Date().toISOString(),
        });

        return {
          success: true,
          workflowType,
          workflowId,
          status: expectedStatus,
          attempts,
          durationMs: duration,
        };
      }

      // Check for terminal failure states
      if (status.status === 'FAILED' || status.status === 'TERMINATED' || status.status === 'CANCELED') {
        return {
          success: false,
          workflowType,
          workflowId,
          expectedStatus,
          actualStatus: status.status,
          reason: 'workflow_terminal_state',
          attempts,
          durationMs: Date.now() - startTime,
        };
      }

      // Sleep before next poll (K6 uses sleep function)
      // Note: In K6, we'd use sleep() but for now we'll just return for the caller to handle
      if (typeof sleep === 'function') {
        sleep(pollIntervalMs / 1000);
      }
    }

    return {
      success: false,
      workflowType,
      workflowId,
      expectedStatus,
      actualStatus: lastStatus,
      reason: 'timeout',
      attempts,
      durationMs: timeoutMs,
    };
  }

  /**
   * Queries a workflow's current state using a query handler
   * @param {string} workflowType - Type of workflow
   * @param {string} queryName - Name of the query
   * @returns {Object} Query result
   */
  queryWorkflowState(workflowType, queryName = 'getState') {
    if (!this.enabled) {
      return { skipped: true };
    }

    const workflowId = this.getWorkflowId(workflowType);
    return {
      workflowType,
      workflowId,
      queryName,
      ...queryWorkflow(workflowId, queryName),
    };
  }

  /**
   * Captures a snapshot of all workflow states
   * @param {string} stageName - Name of the current stage
   * @returns {Object} Workflow snapshot
   */
  captureWorkflowSnapshot(stageName) {
    if (!this.enabled) {
      return { stage: stageName, skipped: true };
    }

    const timestamp = new Date().toISOString();
    const allStatuses = this.getAllWorkflowStatuses();

    const snapshot = {
      stage: stageName,
      timestamp,
      orderId: this.orderId,
      workflows: allStatuses.workflows,
      summary: allStatuses.summary,
    };

    this.timeline.push({
      type: 'snapshot',
      stage: stageName,
      timestamp,
      summary: allStatuses.summary,
    });

    return snapshot;
  }

  /**
   * Logs current workflow states
   * @param {string} context - Context for the log
   */
  logWorkflowStates(context = '') {
    if (!this.enabled || TRACKING_CONFIG.logLevel === 'none') {
      return;
    }

    const statuses = this.getAllWorkflowStatuses();

    console.log(`\n[WORKFLOW-TRACKER] ${context}`);
    console.log(`  Order: ${this.orderId}`);
    console.log(`  Active Workflows:`);

    for (const wf of statuses.workflows) {
      if (wf.found) {
        const statusIcon = wf.isRunning ? '▶' : wf.status === 'COMPLETED' ? '✓' : '✗';
        console.log(`    ${statusIcon} ${wf.workflowType}: ${wf.status || 'unknown'}`);
      }
    }

    console.log(`  Summary: ${statuses.summary.running} running, ${statuses.summary.completed} completed, ${statuses.summary.failed} failed`);
  }

  /**
   * Builds a timeline of workflow execution
   * @returns {Array} Ordered timeline events
   */
  buildWorkflowTimeline() {
    return this.timeline.sort((a, b) =>
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
  }

  /**
   * Generates a comprehensive workflow execution report
   * @returns {Object} Final workflow report
   */
  generateWorkflowReport() {
    if (!this.enabled) {
      return { orderId: this.orderId, trackingEnabled: false };
    }

    const totalDuration = Date.now() - this.startTime;
    const finalStatuses = this.getAllWorkflowStatuses();

    return {
      orderId: this.orderId,
      reportGeneratedAt: new Date().toISOString(),
      totalDuration: {
        ms: totalDuration,
        formatted: this._formatDuration(totalDuration),
      },

      workflowSummary: {
        total: Object.keys(this.workflowIds).length,
        running: finalStatuses.summary.running,
        completed: finalStatuses.summary.completed,
        failed: finalStatuses.summary.failed,
        notStarted: finalStatuses.summary.notFound,
      },

      workflows: finalStatuses.workflows.map(wf => ({
        type: wf.workflowType,
        id: wf.workflowId,
        status: wf.status,
        isRunning: wf.isRunning,
        found: wf.found,
      })),

      signalTracking: {
        totalSignals: this.signalLog.length,
        delivered: this.signalLog.filter(s => s.delivered === true).length,
        unverified: this.signalLog.filter(s => s.verified !== true).length,
        signals: this.signalLog.map(s => ({
          name: s.signalName,
          workflow: s.workflowType,
          sentAt: s.sentAt,
          delivered: s.delivered,
          verified: s.verified,
        })),
      },

      statusHistory: this.statusHistory.map(h => ({
        timestamp: h.timestamp,
        running: h.statuses.filter(s => s.isRunning).length,
        completed: h.statuses.filter(s => s.status === 'COMPLETED').length,
      })),

      timeline: this.buildWorkflowTimeline(),

      workflowDetails: this._getWorkflowDetails(),
    };
  }

  // ==========================================================================
  // PRIVATE HELPER METHODS
  // ==========================================================================

  _getWorkflowDetails() {
    const details = {};

    for (const [type, id] of Object.entries(this.workflowIds)) {
      const description = getWorkflowDescription(id);
      if (description.found) {
        details[type] = {
          workflowId: id,
          status: description.status,
          startTime: description.startTime,
          closeTime: description.closeTime,
          executionTime: description.executionTime,
        };
      }
    }

    return details;
  }

  _formatDuration(ms) {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  }
}

/**
 * Creates a new WorkflowTracker for an order
 * @param {string} orderId - The order ID to track
 * @returns {WorkflowTracker} Tracker instance
 */
export function createWorkflowTracker(orderId) {
  return new WorkflowTracker(orderId);
}

/**
 * Combined tracker that includes both entity and workflow tracking
 * @param {string} orderId - The order ID to track
 * @returns {Object} Combined tracker with both entity and workflow tracking
 */
export function createCombinedTracker(orderId) {
  // Import dynamically to avoid circular deps
  const { EntityStateTracker } = require('./entity-tracker.js');

  return {
    orderId,
    entity: new EntityStateTracker(orderId),
    workflow: new WorkflowTracker(orderId),

    captureFullSnapshot(stageName) {
      return {
        stage: stageName,
        entity: this.entity.captureSnapshot(stageName),
        workflow: this.workflow.captureWorkflowSnapshot(stageName),
      };
    },

    generateFullReport() {
      return {
        orderId,
        entityReport: this.entity.generateReport(),
        workflowReport: this.workflow.generateWorkflowReport(),
      };
    },
  };
}
