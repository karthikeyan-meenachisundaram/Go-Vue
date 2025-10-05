<template>
  <div class="page-container">
    <h1>Employee Details</h1>

    <div v-if="loading" class="status">Loading...</div>
    <div v-else-if="error" class="status error">Error: {{ error }}</div>

    <table v-else-if="employees.length" class="employees-table">
      <thead>
        <tr>
          <th>Employee ID</th>
          <th>Employee Name</th>
          <th>Department</th>
          <th>Language</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="emp in employees" :key="String(emp.emp_id ?? emp.Emp_Id) + String(emp.emp_name ?? emp.EmpName)">
          <td>{{ emp.emp_id ?? emp.Emp_Id ?? '-' }}</td>
          <td>{{ emp.emp_name ?? emp.EmpName ?? '-' }}</td>
          <td>{{ emp.department ?? emp.Department ?? '-' }}</td>     
          <td>{{ emp.language ?? emp.Language ?? '-' }}</td>
          <td class="actions">
            <button @click="goToUpdate(emp)" class="icon-btn edit">
                <i class="fas fa-edit"></i>
            </button>
            <button @click="confirmDelete(emp)" class="icon-btn delete">
                 <i class="fas fa-trash"></i>
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="status">No employees found.</div>
    <!-- BACK TO HOME BUTTON -->
    <button class="back-btn" @click="goHome"> Back to Home</button>
    <!-- Delete Confirmation Dialog -->
    <div v-if="showDialog" class="dialog-overlay">
      <div class="dialog-box">
        <p>Are you sure you want to delete employee <b>{{ selectedEmp?.emp_name }}</b>?</p>
        <div class="dialog-actions">
          <button @click="deleteEmployee" class="delete-btn">Delete</button>
          <button @click="cancelDelete" class="cancel-btn">Cancel</button>
        </div>
      </div>
    </div>

    <!-- Success popup -->
    <div v-if="successMessage" class="success-popup">
      {{ successMessage }}
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';
import { useRouter } from 'vue-router';

const employees = ref([]);
const loading = ref(true);
const error = ref(null);
const router = useRouter();

const showDialog = ref(false);
const selectedEmp = ref(null);
const successMessage = ref('');

function goHome(){
    router.push('/');
}

function goToUpdate(emp) {
  // prefer emp.emp_id but fallback to Emp_Id or EmpID variants
  const id = emp.emp_id ?? emp.Emp_Id ?? emp.EmpID ?? emp.Id;
  if (!id) {
    // optionally show a message instead of navigating
    console.warn('employee has no id, cannot navigate to update');
    return;
  }
  // ensure id is string/number
  router.push(`/update/${id}`);
}

function confirmDelete(emp) {
  selectedEmp.value = emp;
  showDialog.value = true;
}

function cancelDelete() {
  selectedEmp.value = null;
  showDialog.value = false;
}

/*
// change if your backend is on different host/port
const API_URL = 'http://localhost:8080/api/employees';*/

async function deleteEmployee() {
  if (!selectedEmp.value) return;

  try {
    await axios.delete(`http://localhost:8080/api/employees/${selectedEmp.value.emp_id}`);
    employees.value = employees.value.filter(e => e.emp_id !== selectedEmp.value.emp_id);

    successMessage.value = `Employee ${selectedEmp.value.emp_name} deleted successfully`;
    setTimeout(() => successMessage.value = '', 3000);
  } catch (err) {
    console.error('Delete failed:', err);
    alert('Failed to delete employee');
  } finally {
    cancelDelete();
  }
}

// API call to fetch employees
async function fetchEmployees() {
  loading.value = true;
  error.value = null;
  employees.value = [];

  try {
    const res = await axios.get('http://localhost:8080/api/employees');
    employees.value = Array.isArray(res.data) ? res.data : [];
  } catch (err) {
    error.value = err.response?.data?.error || err.message || 'Failed to load employees';
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  fetchEmployees();
});
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100vh;
  padding: 40px 16px;   /* already there, but we’ll use this for spacing */
}

h1 {
  margin: 0 0 24px 0;  /* add bottom margin for space between title & table */
  font-size: 28px;     /* bigger heading */
  text-align: center;
}
.status { margin: 12px 0; }
.status.error { color: #b00020; }

.employees-table {
  margin-top: 20px;     /* space between heading and table */
  width: 95%;
  max-width: 1100px;
  border-collapse: collapse;
  box-shadow: 0 2px 6px rgba(0,0,0,0.06);
}
.employees-table th, .employees-table td {
  border: 1px solid #e0e0e0;
  padding: 8px 16px;
  text-align: left;
  white-space: nowrap;      /* prevent wrapping into multiple lines */
  overflow: hidden;
  text-overflow: ellipsis;
}
.employees-table thead th { 
    background: #f5f5f5; 
    font-weight: 600;
    font-size: 16px; 
}
.employees-table tbody tr:nth-child(even) { 
    background: #fbfbfb; 
}

.back-btn {
  margin-top: 20px;
  padding: 8px 16px;
  background-color: #1976d2;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.back-btn:hover {
  background-color: #1565c0;
}

.actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 4px;
  font-size: 18px;
  margin-right: 6px;
}

.icon-btn.edit { color: #1976d2; }
.icon-btn.delete { color: #d32f2f; }    
.icon-btn:hover {
  opacity: 0.7;
}    

.dialog-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
}

.dialog-box {
  background: white;
  padding: 20px;
  border-radius: 8px;
  min-width: 300px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}

.delete-btn {
  background: #e53935;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 4px;
}

.cancel-btn {
  background: #ccc;
  border: none;
  padding: 6px 12px;
  border-radius: 4px;
}

.success-popup {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: #4caf50;
  color: white;
  padding: 10px 16px;
  border-radius: 4px;
}

</style>