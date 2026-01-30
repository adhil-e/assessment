const asyncHandler = require("express-async-handler");
const { getAllStudents, addNewStudent, getStudentDetail, setStudentStatus, updateStudent } = require("./students-service");

const handleGetAllStudents = asyncHandler(async (req, res) => {
    const { name, className, sectionName, roll } = req.query;
    const students = await getAllStudents({ name, className, sectionName, roll });
    res.json(students);
});

const handleAddStudent = asyncHandler(async (req, res) => {
    const payload = req.body;
    const result = await addNewStudent(payload);
    res.status(201).json(result);
});

const handleUpdateStudent = asyncHandler(async (req, res) => {
    const { id } = req.params;
    const payload = { ...req.body, id };
    const result = await updateStudent(payload);
    res.json(result);
});

const handleGetStudentDetail = asyncHandler(async (req, res) => {
    const { id } = req.params;
    const student = await getStudentDetail(id);
    res.json(student);
});

const handleStudentStatus = asyncHandler(async (req, res) => {
    const { id } = req.params;
    const { status } = req.body;
    const reviewerId = req.user.id;
    const result = await setStudentStatus({ userId: id, reviewerId, status });
    res.json(result);
});

module.exports = {
    handleGetAllStudents,
    handleGetStudentDetail,
    handleAddStudent,
    handleStudentStatus,
    handleUpdateStudent,
};
