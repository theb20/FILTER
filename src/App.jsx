import React from "react";
import { Routes, Route } from "react-router-dom";
import Header from "./assets/components/Header/header.jsx";
import Footer from "./assets/components/Header/Footer.jsx";

function App() {
  return (
    <div className="min-h-screen relative h-full">
      <Header />
      <Footer />
    </div>
  );
}
export default App;
