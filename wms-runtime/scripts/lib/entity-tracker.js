// K6 Entity State Tracker
// Captures and tracks entity state changes across simulation stages

import {
  TRACKING_CONFIG,
  captureOrderState,
  logTrackingState,
  logStateChange,
  getOrderDetails,
  getUnitsForOrder,
  getInventoryForOrder,
  getShipmentForOrder,
  queryOrderFulfillmentState,
} from './tracking.js';

/**
 * EntityStateTracker - Tracks entity state across simulation stages
 * Captures snapshots, computes diffs, and generates reports
 */
export class EntityStateTracker {
  constructor(orderId) {
    this.orderId = orderId;
    this.snapshots = [];
    this.timeline = [];
    this.startTime = Date.now();
    this.enabled = TRACKING_CONFIG.enabled;
  }

  /**
   * Captures a complete state snapshot at a named stage
   * @param {string} stageName - Name of the current stage
   * @returns {Object} Snapshot object
   */
  captureSnapshot(stageName) {
    if (!this.enabled) {
      return { stage: stageName, skipped: true };
    }

    const timestamp = new Date().toISOString();
    const state = captureOrderState(this.orderId);

    const snapshot = {
      stage: stageName,
      timestamp: timestamp,
      capturedAt: Date.now(),
      orderId: this.orderId,
      workflow: state.workflow,
      order: state.order,
      units: state.units,
      inventory: state.inventory,
      shipment: state.shipment,
    };

    this.snapshots.push(snapshot);
    this.timeline.push({
      type: 'snapshot',
      stage: stageName,
      timestamp: timestamp,
    });

    // Log if enabled
    logTrackingState(stageName, state);

    return snapshot;
  }

  /**
   * Captures state before executing a stage
   * @param {string} stageName - Stage name
   * @returns {Object} Before snapshot
   */
  captureBeforeStage(stageName) {
    return this.captureSnapshot(`before_${stageName}`);
  }

  /**
   * Captures state after executing a stage
   * @param {string} stageName - Stage name
   * @returns {Object} After snapshot
   */
  captureAfterStage(stageName) {
    return this.captureSnapshot(`after_${stageName}`);
  }

  /**
   * Compares two snapshots and returns the differences
   * @param {Object} before - Previous snapshot
   * @param {Object} after - Current snapshot
   * @returns {Object} Diff object
   */
  compareSnapshots(before, after) {
    if (!this.enabled || before.skipped || after.skipped) {
      return { skipped: true };
    }

    const diff = {
      fromStage: before.stage,
      toStage: after.stage,
      durationMs: after.capturedAt - before.capturedAt,

      workflow: {
        progressDelta: (after.workflow?.completionPercent || 0) - (before.workflow?.completionPercent || 0),
        stageChanged: before.workflow?.currentStage !== after.workflow?.currentStage,
        previousStage: before.workflow?.currentStage,
        currentStage: after.workflow?.currentStage,
        statusChanged: before.workflow?.status !== after.workflow?.status,
      },

      order: {
        statusChanged: before.order?.status !== after.order?.status,
        previousStatus: before.order?.status,
        currentStatus: after.order?.status,
        waveAssigned: !before.order?.waveId && !!after.order?.waveId,
      },

      units: {
        countDelta: (after.units?.total || 0) - (before.units?.total || 0),
        movementDelta: (after.units?.totalMovements || 0) - (before.units?.totalMovements || 0),
        statusChanges: this._computeStatusChanges(before.units?.byStatus, after.units?.byStatus),
      },

      inventory: {
        reservationDelta: (after.inventory?.reservationCount || 0) - (before.inventory?.reservationCount || 0),
        reservedDelta: (after.inventory?.totalReserved || 0) - (before.inventory?.totalReserved || 0),
      },

      shipment: {
        statusChanged: before.shipment?.status !== after.shipment?.status,
        previousStatus: before.shipment?.status,
        currentStatus: after.shipment?.status,
        trackingAssigned: !before.shipment?.trackingNumber && !!after.shipment?.trackingNumber,
      },
    };

    // Log the change
    logStateChange(before.stage, after.stage, {
      workflow: before.workflow || {},
      order: before.order || {},
      units: before.units || {},
      inventory: before.inventory || {},
      shipment: before.shipment || {},
    }, {
      workflow: after.workflow || {},
      order: after.order || {},
      units: after.units || {},
      inventory: after.inventory || {},
      shipment: after.shipment || {},
    });

    this.timeline.push({
      type: 'diff',
      fromStage: before.stage,
      toStage: after.stage,
      timestamp: after.timestamp,
      changes: this._summarizeChanges(diff),
    });

    return diff;
  }

  /**
   * Tracks a complete stage (before + after + diff)
   * @param {string} stageName - Stage name
   * @param {Function} stageExecutor - Function that executes the stage
   * @returns {Object} Stage tracking result
   */
  trackStage(stageName, stageExecutor) {
    const before = this.captureBeforeStage(stageName);

    let stageResult;
    let stageError = null;
    const stageStartTime = Date.now();

    try {
      stageResult = stageExecutor();
    } catch (e) {
      stageError = e.message || 'Unknown error';
    }

    const stageDuration = Date.now() - stageStartTime;
    const after = this.captureAfterStage(stageName);
    const diff = this.compareSnapshots(before, after);

    return {
      stage: stageName,
      success: stageError === null,
      error: stageError,
      stageResult,
      stageDuration,
      diff,
      before,
      after,
    };
  }

  /**
   * Gets the most recent snapshot
   * @returns {Object|null} Latest snapshot
   */
  getLatestSnapshot() {
    return this.snapshots.length > 0 ? this.snapshots[this.snapshots.length - 1] : null;
  }

  /**
   * Gets all snapshots
   * @returns {Array} All snapshots
   */
  getAllSnapshots() {
    return this.snapshots;
  }

  /**
   * Generates a comprehensive tracking report
   * @returns {Object} Final report
   */
  generateReport() {
    if (!this.enabled) {
      return { orderId: this.orderId, trackingEnabled: false };
    }

    const totalDuration = Date.now() - this.startTime;
    const firstSnapshot = this.snapshots[0];
    const lastSnapshot = this.snapshots[this.snapshots.length - 1];

    return {
      orderId: this.orderId,
      reportGeneratedAt: new Date().toISOString(),
      totalDuration: {
        ms: totalDuration,
        formatted: this._formatDuration(totalDuration),
      },

      summary: {
        stagesTracked: this.snapshots.length,
        timelineEvents: this.timeline.length,
        initialWorkflowStage: firstSnapshot?.workflow?.currentStage,
        finalWorkflowStage: lastSnapshot?.workflow?.currentStage,
        initialOrderStatus: firstSnapshot?.order?.status,
        finalOrderStatus: lastSnapshot?.order?.status,
        finalUnitCount: lastSnapshot?.units?.total || 0,
        finalShipmentStatus: lastSnapshot?.shipment?.status,
      },

      stageBreakdown: this.snapshots.map((snapshot, index) => ({
        stage: snapshot.stage,
        timestamp: snapshot.timestamp,
        workflow: {
          stage: snapshot.workflow?.currentStage,
          progress: snapshot.workflow?.completionPercent,
          status: snapshot.workflow?.status,
        },
        order: {
          status: snapshot.order?.status,
          itemCount: snapshot.order?.itemCount,
        },
        units: {
          total: snapshot.units?.total,
          byStatus: snapshot.units?.byStatus,
        },
        shipment: {
          status: snapshot.shipment?.status,
        },
      })),

      entityLifecycles: {
        order: this._extractOrderLifecycle(),
        units: this._extractUnitLifecycle(),
        inventory: this._extractInventoryLifecycle(),
        shipment: this._extractShipmentLifecycle(),
      },

      timeline: this.timeline,
    };
  }

  // ==========================================================================
  // PRIVATE HELPER METHODS
  // ==========================================================================

  _computeStatusChanges(beforeStatus, afterStatus) {
    const changes = {};
    const allStatuses = new Set([
      ...Object.keys(beforeStatus || {}),
      ...Object.keys(afterStatus || {}),
    ]);

    for (const status of allStatuses) {
      const beforeCount = (beforeStatus || {})[status] || 0;
      const afterCount = (afterStatus || {})[status] || 0;
      if (beforeCount !== afterCount) {
        changes[status] = {
          before: beforeCount,
          after: afterCount,
          delta: afterCount - beforeCount,
        };
      }
    }

    return changes;
  }

  _summarizeChanges(diff) {
    const changes = [];

    if (diff.workflow?.stageChanged) {
      changes.push(`workflow: ${diff.workflow.previousStage} → ${diff.workflow.currentStage}`);
    }
    if (diff.workflow?.progressDelta > 0) {
      changes.push(`progress: +${diff.workflow.progressDelta}%`);
    }
    if (diff.order?.statusChanged) {
      changes.push(`order: ${diff.order.previousStatus} → ${diff.order.currentStatus}`);
    }
    if (diff.units?.movementDelta > 0) {
      changes.push(`movements: +${diff.units.movementDelta}`);
    }
    if (diff.shipment?.statusChanged) {
      changes.push(`shipment: ${diff.shipment.previousStatus} → ${diff.shipment.currentStatus}`);
    }

    return changes;
  }

  _extractOrderLifecycle() {
    const statusChanges = [];
    let previousStatus = null;

    for (const snapshot of this.snapshots) {
      const currentStatus = snapshot.order?.status;
      if (currentStatus && currentStatus !== previousStatus) {
        statusChanges.push({
          stage: snapshot.stage,
          status: currentStatus,
          timestamp: snapshot.timestamp,
        });
        previousStatus = currentStatus;
      }
    }

    return {
      statusChanges,
      totalTransitions: statusChanges.length,
    };
  }

  _extractUnitLifecycle() {
    const transitions = [];
    let previousTotal = 0;

    for (const snapshot of this.snapshots) {
      const currentTotal = snapshot.units?.total || 0;
      const currentByStatus = snapshot.units?.byStatus || {};

      if (currentTotal !== previousTotal || Object.keys(currentByStatus).length > 0) {
        transitions.push({
          stage: snapshot.stage,
          total: currentTotal,
          byStatus: currentByStatus,
          timestamp: snapshot.timestamp,
        });
      }
      previousTotal = currentTotal;
    }

    return {
      transitions,
      finalCount: previousTotal,
    };
  }

  _extractInventoryLifecycle() {
    const changes = [];
    let previousReserved = 0;

    for (const snapshot of this.snapshots) {
      const currentReserved = snapshot.inventory?.totalReserved || 0;

      if (currentReserved !== previousReserved) {
        changes.push({
          stage: snapshot.stage,
          totalReserved: currentReserved,
          delta: currentReserved - previousReserved,
          timestamp: snapshot.timestamp,
        });
      }
      previousReserved = currentReserved;
    }

    return {
      changes,
      finalReserved: previousReserved,
    };
  }

  _extractShipmentLifecycle() {
    const statusChanges = [];
    let previousStatus = null;

    for (const snapshot of this.snapshots) {
      const currentStatus = snapshot.shipment?.status;
      if (currentStatus && currentStatus !== previousStatus) {
        statusChanges.push({
          stage: snapshot.stage,
          status: currentStatus,
          trackingNumber: snapshot.shipment?.trackingNumber,
          timestamp: snapshot.timestamp,
        });
        previousStatus = currentStatus;
      }
    }

    return {
      statusChanges,
      totalTransitions: statusChanges.length,
    };
  }

  _formatDuration(ms) {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  }
}

/**
 * Creates a new EntityStateTracker for an order
 * @param {string} orderId - The order ID to track
 * @returns {EntityStateTracker} Tracker instance
 */
export function createEntityTracker(orderId) {
  return new EntityStateTracker(orderId);
}
