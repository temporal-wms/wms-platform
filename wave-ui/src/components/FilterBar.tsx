interface FilterBarProps {
  priorityFilter: string[];
  onPriorityChange: (priorities: string[]) => void;
  stateFilter: string[];
  onStateChange: (states: string[]) => void;
  availableStates: string[];
}

const PRIORITIES = [
  { value: 'same_day', label: 'Same Day' },
  { value: 'next_day', label: 'Next Day' },
  { value: 'standard', label: 'Standard' },
];

export function FilterBar({
  priorityFilter,
  onPriorityChange,
  stateFilter,
  onStateChange,
  availableStates,
}: FilterBarProps) {
  const handlePriorityToggle = (priority: string) => {
    if (priorityFilter.includes(priority)) {
      onPriorityChange(priorityFilter.filter(p => p !== priority));
    } else {
      onPriorityChange([...priorityFilter, priority]);
    }
  };

  const handleStateChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedOptions = Array.from(e.target.selectedOptions, option => option.value);
    onStateChange(selectedOptions);
  };

  const handleClearFilters = () => {
    onPriorityChange([]);
    onStateChange([]);
  };

  const hasActiveFilters = priorityFilter.length > 0 || stateFilter.length > 0;

  return (
    <div className="filter-bar">
      <div className="filter-group">
        <label className="filter-label">Priority:</label>
        <div className="filter-checkboxes">
          {PRIORITIES.map(({ value, label }) => (
            <label key={value} className="filter-checkbox">
              <input
                type="checkbox"
                checked={priorityFilter.includes(value)}
                onChange={() => handlePriorityToggle(value)}
              />
              <span className={`priority-tag priority-${value}`}>{label}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="filter-group">
        <label className="filter-label">Ship To State:</label>
        <select
          className="filter-select"
          multiple
          value={stateFilter}
          onChange={handleStateChange}
          size={1}
        >
          {availableStates.map(state => (
            <option key={state} value={state}>
              {state}
            </option>
          ))}
        </select>
        {stateFilter.length > 0 && (
          <span className="selected-states">
            {stateFilter.join(', ')}
          </span>
        )}
      </div>

      {hasActiveFilters && (
        <button className="clear-filters-btn" onClick={handleClearFilters}>
          Clear Filters
        </button>
      )}
    </div>
  );
}
