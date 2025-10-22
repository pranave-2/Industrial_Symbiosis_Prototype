import os
import re
import PyPDF2
import docx
from typing import Dict, List, Any
import json

class DocumentParser:
    """Parse documents and extract industrial profile information"""
    
    def __init__(self):
        self.input_keywords = [
            'raw material', 'input', 'resource', 'feed', 'consume',
            'material', 'ingredient', 'supply', 'purchase'
        ]
        self.output_keywords = [
            'output', 'product', 'waste', 'byproduct', 'by-product',
            'residue', 'emission', 'discharge', 'scrap', 'slag'
        ]
        
    def parse(self, file_path: str) -> Dict[str, Any]:
        """Parse document based on file extension"""
        ext = os.path.splitext(file_path)[1].lower()
        
        if ext == '.pdf':
            text = self._parse_pdf(file_path)
        elif ext == '.docx':
            text = self._parse_docx(file_path)
        elif ext == '.txt':
            text = self._parse_txt(file_path)
        else:
            raise ValueError(f"Unsupported file type: {ext}")
        
        return self._extract_profile(text)
    
    def _parse_pdf(self, file_path: str) -> str:
        """Extract text from PDF"""
        text = ""
        with open(file_path, 'rb') as file:
            reader = PyPDF2.PdfReader(file)
            for page in reader.pages:
                text += page.extract_text() + "\n"
        return text
    
    def _parse_docx(self, file_path: str) -> str:
        """Extract text from DOCX"""
        doc = docx.Document(file_path)
        text = "\n".join([para.text for para in doc.paragraphs])
        return text
    
    def _parse_txt(self, file_path: str) -> str:
        """Extract text from TXT"""
        with open(file_path, 'r', encoding='utf-8') as file:
            return file.read()
    
    def _extract_profile(self, text: str) -> Dict[str, Any]:
        """Extract structured profile from text"""
        
        # Extract company name (look for common patterns)
        name = self._extract_company_name(text)
        
        # Extract location
        location = self._extract_location(text)
        
        # Extract inputs
        inputs = self._extract_items(text, self.input_keywords)
        
        # Extract outputs
        outputs = self._extract_outputs(text)
        
        return {
            "name": name,
            "location": location,
            "inputs": inputs,
            "outputs": outputs
        }
    
    def _extract_company_name(self, text: str) -> str:
        """Extract company name from text"""
        lines = text.split('\n')
        
        # Look for patterns like "Company:", "Industry:", or first capitalized line
        for line in lines[:10]:  # Check first 10 lines
            line = line.strip()
            if not line:
                continue
                
            # Check for explicit labels
            if re.match(r'(company|industry|name|organization)[:]\s*(.+)', line, re.IGNORECASE):
                match = re.match(r'(company|industry|name|organization)[:]\s*(.+)', line, re.IGNORECASE)
                return match.group(2).strip()
            
            # Use first substantial capitalized line
            if len(line) > 5 and line[0].isupper():
                return line
        
        return "Unknown Company"
    
    def _extract_location(self, text: str) -> Dict[str, float]:
        """Extract location from text"""
        # Look for coordinates
        coord_pattern = r'(-?\d+\.?\d*)\s*[,°]\s*(-?\d+\.?\d*)'
        coords = re.search(coord_pattern, text)
        
        if coords:
            return {
                "lat": float(coords.group(1)),
                "lng": float(coords.group(2))
            }
        
        # Look for city names (simplified - just return default)
        # In production, you'd use geocoding API
        city_pattern = r'(location|city|address)[:]\s*([^.\n]+)'
        city = re.search(city_pattern, text, re.IGNORECASE)
        
        # Default location (can be enhanced with geocoding)
        return {"lat": 0.0, "lng": 0.0}
    
    def _extract_items(self, text: str, keywords: List[str]) -> List[str]:
        """Extract items based on keywords"""
        items = []
        lines = text.split('\n')
        
        # Look for sections with keywords
        in_section = False
        for i, line in enumerate(lines):
            line = line.strip()
            
            # Check if line contains keyword
            for keyword in keywords:
                if keyword.lower() in line.lower():
                    in_section = True
                    # Extract items from this line and next few
                    items.extend(self._extract_list_items(lines[i:min(i+10, len(lines))]))
                    break
            
            # Stop if we hit another major section
            if in_section and any(kw in line.lower() for kw in ['waste', 'output', 'process']):
                if not any(kw in line.lower() for kw in keywords):
                    break
        
        # Clean and deduplicate
        items = list(set([self._clean_item(item) for item in items if item]))
        return items[:10]  # Limit to 10 items
    
    def _extract_outputs(self, text: str) -> List[Dict[str, Any]]:
        """Extract output/waste streams with details"""
        outputs = []
        lines = text.split('\n')
        
        for i, line in enumerate(lines):
            line = line.strip()
            
            # Look for output/waste keywords
            for keyword in self.output_keywords:
                if keyword.lower() in line.lower():
                    # Try to extract details
                    output = self._parse_output_line(line)
                    if output:
                        outputs.append(output)
                    
                    # Check next few lines for additional details
                    for j in range(1, min(5, len(lines) - i)):
                        next_line = lines[i + j].strip()
                        if next_line:
                            additional = self._parse_output_line(next_line)
                            if additional:
                                outputs.append(additional)
        
        # Deduplicate by name
        seen = set()
        unique_outputs = []
        for output in outputs:
            if output['name'] not in seen:
                seen.add(output['name'])
                unique_outputs.append(output)
        
        return unique_outputs[:10]  # Limit to 10 outputs
    
    def _parse_output_line(self, line: str) -> Dict[str, Any]:
        """Parse a line to extract output details"""
        # Look for patterns like "waste slag (200 tons/month)"
        
        # Extract quantity pattern
        quantity_pattern = r'(\d+\.?\d*)\s*(tons?|kg|litres?|m3|cubic|gallons?)[/\s]*(month|year|day)?'
        quantity_match = re.search(quantity_pattern, line, re.IGNORECASE)
        
        # Extract state
        state = 'solid'  # default
        if re.search(r'\b(liquid|fluid|water|oil)\b', line, re.IGNORECASE):
            state = 'liquid'
        elif re.search(r'\b(gas|vapor|emission|air)\b', line, re.IGNORECASE):
            state = 'gas'
        
        # Extract name (simplified)
        name_pattern = r'([a-zA-Z\s]+(?:slag|waste|residue|scrap|byproduct|emission|ash|sludge))'
        name_match = re.search(name_pattern, line, re.IGNORECASE)
        
        if name_match or quantity_match:
            return {
                "name": name_match.group(1).strip() if name_match else self._clean_item(line),
                "state": state,
                "quantity": quantity_match.group(0) if quantity_match else "Unknown"
            }
        
        return None
    
    def _extract_list_items(self, lines: List[str]) -> List[str]:
        """Extract items from list-like text"""
        items = []
        
        for line in lines:
            line = line.strip()
            if not line:
                continue
            
            # Check for bullet points or numbered lists
            if re.match(r'^[\-\*•●◦▪▫]\s*(.+)', line):
                match = re.match(r'^[\-\*•●◦▪▫]\s*(.+)', line)
                items.append(match.group(1).strip())
            elif re.match(r'^\d+[\.)]\s*(.+)', line):
                match = re.match(r'^\d+[\.)]\s*(.+)', line)
                items.append(match.group(1).strip())
            elif ',' in line and len(line.split(',')) > 1:
                # Comma-separated list
                items.extend([item.strip() for item in line.split(',')])
        
        return items
    
    def _clean_item(self, item: str) -> str:
        """Clean extracted item text"""
        # Remove special characters, extra spaces
        item = re.sub(r'[^\w\s\-/()]', '', item)
        item = ' '.join(item.split())
        return item.lower()[:100]  # Limit length

if __name__ == "__main__":
    # Test parser
    parser = DocumentParser()
    test_text = """
    Steel Rolling Mill A
    Location: 12.34, 56.78
    
    Inputs:
    - scrap metal
    - coal
    - cooling water
    
    Outputs:
    - waste slag (200 tons/month)
    - cooling water discharge (liquid, 500 litres/day)
    """
    
    profile = parser._extract_profile(test_text)
    print(json.dumps(profile, indent=2))