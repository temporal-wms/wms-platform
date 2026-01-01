import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { WaveList } from './components/WaveList';
import { WaveDetails } from './components/WaveDetails';
import { CreateWave } from './components/CreateWave';

function App() {
  return (
    <div className="waves-app">
      <Routes>
        <Route path="/" element={<WaveList />} />
        <Route path="/new" element={<CreateWave />} />
        <Route path="/:waveId" element={<WaveDetails />} />
      </Routes>
    </div>
  );
}

export default App;
