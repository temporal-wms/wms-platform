import React, { useState } from 'react';
import { Routes, Route } from 'react-router-dom';
import { StowTaskList } from './components/StowTaskList';
import { StowTaskDetails } from './components/StowTaskDetails';
import { WorkerTaskAssignment } from './components/WorkerTaskAssignment';
import { TaskFilterPanel } from './components/TaskFilterPanel';

function StowApp() {
  return (
    <div className="stow">
      <Routes>
        <Route path="/" element={<StowTaskList />} />
        <Route path="/:taskId" element={<StowTaskDetails />} />
      </Routes>
    </div>
  );
}

export default StowApp;
