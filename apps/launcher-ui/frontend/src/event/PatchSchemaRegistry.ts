/**
 * PatchSchemaRegistry
 * Centralized schema definitions for LauncherEventPatch validation.
 * Single source of truth for allowed patch keys and constraints per domain.
 */

export type AllowedPatchKey<T> = keyof T;

export interface ValidationRule {
  required?: boolean;
  type: string | string[];
  minLength?: number;
  maxLength?: number;
  minValue?: number;
  maxValue?: number;
  allowNull?: boolean;
  oneOf?: unknown[];
  custom?: (value: unknown) => boolean | string;
}

export interface DomainSchema {
  [key: string]: ValidationRule;
}

export class PatchSchemaRegistry {
  private static readonly TRACE_NODE_TYPES = ['resolve', 'download', 'verify', 'install', 'complete', 'error'] as const;
  private static readonly LAUNCH_STATES = [
    'idle',
    'resolving',
    'downloading',
    'installing',
    'verifying',
    'materializing',
    'launching',
    'running',
    'complete',
    'error',
  ] as const;

  // Schema for trace patch domain
  static readonly traceNodeSchema: DomainSchema = {
    type: { type: 'string', oneOf: [...this.TRACE_NODE_TYPES] },
    modId: { type: 'string', minLength: 1 },
    modName: { type: 'string', minLength: 1 },
    state: { type: 'string', required: false },
    error: { type: 'string', required: false, allowNull: true },
    progress: {
      type: 'object',
      required: false,
      custom: (val: unknown) => {
        if (!val) return true;
        if (typeof val !== 'object' || !val) return 'progress must be an object';
        const p = val as any;
        if (typeof p.current !== 'number') return 'progress.current must be number';
        if (typeof p.total !== 'number') return 'progress.total must be number';
        if (typeof p.percent !== 'number') return 'progress.percent must be number';
        if (p.speed !== undefined && typeof p.speed !== 'number') return 'progress.speed must be number';
        if (p.eta !== undefined && typeof p.eta !== 'number') return 'progress.eta must be number';
        return true;
      },
    },
  };

  static readonly tracePatchSchema: DomainSchema = {
    addEvents: { type: 'array', required: false },
    updateEventProgress: { type: 'object', required: false },
    completeNode: { type: 'object', required: false },
    activeTrace: { type: ['string', 'null'], required: false, allowNull: true },
  };

  static readonly downloadsPatchSchema: DomainSchema = {
    sessionUpdate: { type: 'object', required: false },
    completeSessionId: { type: 'string', required: false },
    failSession: { type: 'object', required: false },
  };

  static readonly sessionPatchSchema: DomainSchema = {
    currentSessionId: { type: ['string', 'null'], required: false, allowNull: true },
    launchState: { type: 'string', oneOf: [...this.LAUNCH_STATES], required: false },
    currentServer: { type: ['object', 'null'], required: false, allowNull: true },
    lastError: { type: ['string', 'null'], required: false, allowNull: true },
    resetSession: { type: 'boolean', required: false },
  };

  static readonly serversPatchSchema: DomainSchema = {
    joining: { type: 'boolean', required: false },
  };

  /**
   * Get allowed keys for a domain
   */
  static getAllowedKeys(domain: 'trace' | 'downloads' | 'session' | 'servers'): string[] {
    const schemas = {
      trace: this.tracePatchSchema,
      downloads: this.downloadsPatchSchema,
      session: this.sessionPatchSchema,
      servers: this.serversPatchSchema,
    };
    return Object.keys(schemas[domain]);
  }

  /**
   * Get schema for a domain
   */
  static getSchema(domain: 'trace' | 'downloads' | 'session' | 'servers'): DomainSchema {
    const schemas = {
      trace: this.tracePatchSchema,
      downloads: this.downloadsPatchSchema,
      session: this.sessionPatchSchema,
      servers: this.serversPatchSchema,
    };
    return schemas[domain];
  }

  /**
   * Validate a value against a specific domain schema
   */
  static validateAgainstSchema(domain: 'trace' | 'downloads' | 'session' | 'servers', value: unknown): string[] {
    const errors: string[] = [];
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
      return [`${domain} patch must be an object`];
    }

    const schema = this.getSchema(domain);
    const allowedKeys = new Set(Object.keys(schema));
    const obj = value as Record<string, unknown>;

    // Check for unexpected keys
    Object.keys(obj).forEach((key) => {
      if (!allowedKeys.has(key)) {
        errors.push(`${domain} patch contains unexpected key: ${key}`);
      }
    });

    // Validate each field against its schema
    Object.entries(obj).forEach(([key, fieldValue]) => {
      const rule = schema[key];
      if (!rule) return;

      const fieldErrors = this.validateField(fieldValue, rule, `${domain}.${key}`);
      errors.push(...fieldErrors);
    });

    return errors;
  }

  /**
   * Validate a single field value against a rule
   */
  private static validateField(value: unknown, rule: ValidationRule, path: string): string[] {
    const errors: string[] = [];

    // Check null allowance
    if (value === null) {
      if (!rule.allowNull) {
        errors.push(`${path} cannot be null`);
      }
      return errors;
    }

    // Check custom validation first
    if (rule.custom) {
      const customResult = rule.custom(value);
      if (customResult !== true) {
        errors.push(`${path}: ${typeof customResult === 'string' ? customResult : 'failed custom validation'}`);
        return errors;
      }
    }

    // Check type
    const types = Array.isArray(rule.type) ? rule.type : [rule.type];
    const valueType = Array.isArray(value) ? 'array' : typeof value;

    if (!types.includes(valueType)) {
      errors.push(`${path} must be of type ${types.join(' | ')}, got ${valueType}`);
      return errors;
    }

    // Check oneOf
    if (rule.oneOf && !rule.oneOf.includes(value)) {
      errors.push(`${path} must be one of: ${rule.oneOf.join(', ')}`);
      return errors;
    }

    // String-specific checks
    if (typeof value === 'string') {
      if (rule.minLength !== undefined && value.length < rule.minLength) {
        errors.push(`${path} must be at least ${rule.minLength} characters`);
      }
      if (rule.maxLength !== undefined && value.length > rule.maxLength) {
        errors.push(`${path} must be at most ${rule.maxLength} characters`);
      }
    }

    // Number-specific checks
    if (typeof value === 'number') {
      if (rule.minValue !== undefined && value < rule.minValue) {
        errors.push(`${path} must be at least ${rule.minValue}`);
      }
      if (rule.maxValue !== undefined && value > rule.maxValue) {
        errors.push(`${path} must be at most ${rule.maxValue}`);
      }
    }

    return errors;
  }
}
